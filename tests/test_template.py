import os
import shutil
import socket
import subprocess
import time
import urllib.request
from pathlib import Path

import pytest
import yaml

ROOT = Path(__file__).resolve().parent.parent
FIXTURES_DIR = ROOT / "tests" / "fixtures"
COOKIECUTTER_VERSION = os.environ.get("COOKIECUTTER_VERSION", "2.7.1")
FIXTURES = sorted(path.stem for path in FIXTURES_DIR.glob("*.yaml"))


def run(cmd, *, cwd=None, env=None, check=True):
    result = subprocess.run(
        [str(part) for part in cmd],
        cwd=cwd,
        env=env,
        capture_output=True,
        text=True,
    )
    if check and result.returncode != 0:
        raise AssertionError(
            f"command failed ({result.returncode}): {' '.join(str(part) for part in cmd)}\n"
            f"stdout:\n{result.stdout}\n"
            f"stderr:\n{result.stderr}"
        )
    return result


def generate(fixture_path, output_dir, *, check=True):
    return run(
        [
            "uvx",
            "--from",
            f"cookiecutter=={COOKIECUTTER_VERSION}",
            "cookiecutter",
            "--no-input",
            "--config-file",
            fixture_path,
            "--output-dir",
            output_dir,
            ROOT,
        ],
        check=check,
    )


def read_fixture(fixture_path):
    with open(fixture_path, encoding="utf-8") as handle:
        return yaml.safe_load(handle)["default_context"]


def environment():
    env = os.environ.copy()
    env["STAGE"] = "testing"
    return env


@pytest.fixture(scope="session")
def generated_projects(tmp_path_factory):
    projects = {}
    base = tmp_path_factory.mktemp("generated")

    for fixture in FIXTURES:
        fixture_path = FIXTURES_DIR / f"{fixture}.yaml"
        context = read_fixture(fixture_path)
        output_dir = base / fixture
        output_dir.mkdir()

        generate(fixture_path, output_dir)

        project = output_dir / context["project_slug"]
        assert project.is_dir(), f"no se generó {project}"

        projects[fixture] = {
            "path": project,
            "context": context,
            "fixture": fixture_path,
        }

    return projects


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_project_builds(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    env = environment()

    run(["go", "mod", "download"], cwd=project, env=env)
    run(["go", "mod", "tidy"], cwd=project, env=env)
    tidy_diff = run(["go", "mod", "tidy", "-diff"], cwd=project, env=env, check=False)
    if tidy_diff.returncode != 0:
        if "flag provided but not defined" in tidy_diff.stderr:
            pytest.skip("go mod tidy -diff no está disponible en esta versión de Go")
        raise AssertionError(
            "go mod tidy -diff encontró cambios:\n"
            f"stdout:\n{tidy_diff.stdout}\nstderr:\n{tidy_diff.stderr}"
        )

    run(["go", "build", "./..."], cwd=project, env=env)
    run(["go", "vet", "./..."], cwd=project, env=env)
    run(["go", "test", "./..."], cwd=project, env=env)


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_project_conditional_files(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]

    use_db = context["use_db"] == "yes"
    use_redis = context["use_redis"] == "yes"
    use_asynq = context["use_asynq"] == "yes"
    use_worker = context["use_worker"] == "yes" and use_asynq
    use_swagger = context["use_swagger"] == "yes"
    use_pprof = context["use_pprof"] == "yes"
    use_sentry = context["use_sentry"] == "yes"

    checks = {
        "configs/client_db.go": use_db,
        "cmd/devcli/main.go": use_db,
        "cmd/atlas_loader/main.go": use_db,
        "atlas.hcl": use_db,
        "migrations/.gitkeep": use_db,
        "configs/client_redis.go": use_redis,
        "configs/client_asynq.go": use_asynq,
        "internal/core/asynctasks": use_asynq,
        "cmd/worker/main.go": use_worker,
        "compose.yaml": use_db or use_redis or use_asynq,
        "env/testing.env": True,
        "internal/testutil/context.go": True,
    }

    for relative, expected in checks.items():
        path = project / relative
        assert path.exists() is expected, f"{relative}: existe={path.exists()}, esperado={expected}"

    router = (project / "internal/server/router.go").read_text(encoding="utf-8")
    assert ("ginSwagger" in router) is use_swagger
    assert ("pprof" in router) is use_pprof

    logging = (project / "configs/logging.go").read_text(encoding="utf-8")
    assert ("sentry" in logging) is use_sentry

    if use_db:
        assert "module " in (project / "go.mod").read_text(encoding="utf-8")
        assert "_test" in (project / "env/testing.env").read_text(encoding="utf-8")


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_ci_yaml_is_valid(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    workflow = project / ".github" / "workflows" / "ci.yml"
    assert workflow.exists()

    with open(workflow, encoding="utf-8") as handle:
        data = yaml.safe_load(handle)

    assert "jobs" in data
    assert "test" in data["jobs"]
    assert data["jobs"]["test"]["steps"]


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_compose_is_valid(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    compose = project / "compose.yaml"

    if not compose.exists():
        pytest.skip("el fixture no incluye compose.yaml")
    if not _docker_available(project):
        pytest.skip("docker compose no está disponible")

    run(["docker", "compose", "config", "-q"], cwd=project)


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_project_lints(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    env = environment()

    if shutil.which("golangci-lint") is None:
        pytest.skip("golangci-lint no está instalado")
    run(["golangci-lint", "run"], cwd=project, env=env)

    if shutil.which("govulncheck") is None:
        pytest.skip("govulncheck no está instalado")

    result = run(["govulncheck", "./..."], cwd=project, env=env, check=False)
    if result.returncode != 0:
        detail = (result.stdout + result.stderr)[-4000:]
        if os.environ.get("REQUIRE_GOVULNCHECK") == "1":
            raise AssertionError(f"govulncheck falló:\n{detail}")
        pytest.xfail(
            "govulncheck falla con las dependencias de la Fase 0; "
            "se corrige al actualizar en la Fase 1:\n" + detail
        )


@pytest.mark.parametrize(
    ("project_slug", "go_module", "use_worker", "use_asynq", "message"),
    [
        ("1invalid", "github.com/acme/app", "no", "no", "project_slug debe empezar"),
        ("invalid slug", "github.com/acme/app", "no", "no", "project_slug debe empezar"),
        ("app..x", "github.com/acme/app", "no", "no", "project_slug debe empezar"),
        ("app", "github.com/acme/../app", "no", "no", "go_module no puede contener '..'"),
        ("app", "github.com/acme/app func", "no", "no", "go_module no puede contener espacios"),
        ("app", "github.com/acme/func", "no", "no", "palabra reservada"),
        ("app", "github.com/acme//app", "no", "no", "segmentos vacíos"),
        ("app", "github.com/acme/app", "yes", "no", "use_worker=yes requiere use_asynq=yes"),
    ],
)
def test_pre_gen_rejects_invalid_values(
    tmp_path, project_slug, go_module, use_worker, use_asynq, message
):
    context = {
        "project_name": "Invalid",
        "project_slug": project_slug,
        "description": "invalid fixture",
        "author": "Example",
        "email": "example@example.com",
        "go_module": go_module,
        "license": "MIT",
        "version": "0.1.0",
        "use_db": "no",
        "use_redis": "no",
        "use_asynq": use_asynq,
        "use_worker": use_worker,
        "use_swagger": "no",
        "use_pprof": "no",
        "use_sentry": "no",
    }
    fixture = tmp_path / "invalid.yaml"
    fixture.write_text(yaml.safe_dump({"default_context": context}), encoding="utf-8")

    output_dir = tmp_path / "out"
    output_dir.mkdir()

    result = generate(fixture, output_dir, check=False)

    assert result.returncode != 0, "la generación debía fallar"
    output = result.stdout + result.stderr
    assert message in output, f"no se encontró {message!r} en:\n{output}"


def _port_open(port):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.settimeout(1)
        return sock.connect_ex(("127.0.0.1", port)) == 0


def _wait_for_port(port, timeout=60):
    deadline = time.time() + timeout
    while time.time() < deadline:
        if _port_open(port):
            return True
        time.sleep(1)
    return False


def _wait_for_health(url, timeout=60):
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(url, timeout=2) as response:
                if response.status == 200:
                    return True
        except OSError:
            time.sleep(1)
    return False


def _docker_available(project):
    if shutil.which("docker") is None:
        return False
    result = run(["docker", "compose", "version"], cwd=project, check=False)
    return result.returncode == 0


def test_full_project_http_smoke(generated_projects):
    project = generated_projects["full"]["path"]

    if not _docker_available(project):
        pytest.skip("docker compose no está disponible")
    if _port_open(5432) or _port_open(6379) or _port_open(8080):
        pytest.skip("los puertos 5432/6379/8080 ya están ocupados")

    env = os.environ.copy()
    server = None

    try:
        run(["docker", "compose", "up", "-d"], cwd=project)
        assert _wait_for_port(5432), "Postgres no abrió el puerto"
        assert _wait_for_port(6379), "Redis no abrió el puerto"

        run(["go", "build", "-o", "/tmp/cookiecutter-smoke-http", "./cmd/http"], cwd=project, env=env)
        server = subprocess.Popen(
            ["/tmp/cookiecutter-smoke-http"],
            cwd=project,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
        )

        if not _wait_for_health("http://127.0.0.1:8080/healthz"):
            output = ""
            if server.poll() is not None:
                output = server.stdout.read() if server.stdout else ""
            pytest.fail(f"la API no respondió en /healthz. Salida:\n{output}")
    finally:
        if server is not None and server.poll() is None:
            server.terminate()
            try:
                server.wait(timeout=10)
            except subprocess.TimeoutExpired:
                server.kill()
        run(["docker", "compose", "down", "-v"], cwd=project, check=False)


def test_generated_tasks_and_generators(generated_projects, tmp_path):
    source = generated_projects["full"]["path"]
    project = tmp_path / "project"
    shutil.copytree(source, project)

    run(
        ["uvx", "--from", "invoke", "--with", "jinja2", "inv", "format-env", "--stage", "dev"],
        cwd=project,
    )
    dev_env = (project / "env" / "dev.env").read_text(encoding="utf-8")
    assert "{{" not in dev_env

    run(
        ["uvx", "--from", "invoke", "--with", "jinja2", "inv", "gen-model", "HarnessModel"],
        cwd=project,
    )

    for relative in (
        "internal/core/models/harness_model.go",
        "internal/core/repositories/harness_model_repository.go",
        "internal/core/services/harness_model_service.go",
    ):
        assert (project / relative).exists(), f"no se generó {relative}"

    env = environment()
    run(["go", "mod", "tidy"], cwd=project, env=env)
    run(["go", "build", "./..."], cwd=project, env=env)
    run(["go", "test", "./..."], cwd=project, env=env)


def test_generated_go_module_is_used(generated_projects):
    for fixture, data in generated_projects.items():
        project = data["path"]
        go_module = data["context"]["go_module"]
        go_mod = (project / "go.mod").read_text(encoding="utf-8")
        assert f"module {go_module}" in go_mod, f"{fixture} no usa go_module en go.mod"

        router = (project / "internal/server/router.go").read_text(encoding="utf-8")
        assert go_module in router, f"{fixture} no usa go_module en router.go"

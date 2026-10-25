import json
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
ALL_FIXTURES = sorted(path.stem for path in FIXTURES_DIR.glob("*.yaml"))

# FIXTURES=minimal,full acota la matriz (la usa la CI para PR); sin valor se
# generan los ocho fixtures. Un nombre desconocido es un error, no un salto.
_selected = {name.strip() for name in os.environ.get("FIXTURES", "").split(",") if name.strip()}
_unknown = _selected - set(ALL_FIXTURES)
if _unknown:
    raise AssertionError(
        f"fixtures desconocidos: {sorted(_unknown)}; disponibles: {ALL_FIXTURES}"
    )
FIXTURES = [name for name in ALL_FIXTURES if name in _selected] if _selected else ALL_FIXTURES


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
    if fixture == "full":
        _assert_tidy_clean(project, env)
    run(["go", "mod", "tidy"], cwd=project, env=env)
    _assert_tidy_clean(project, env)

    run(["go", "build", "./..."], cwd=project, env=env)
    run(["go", "vet", "./..."], cwd=project, env=env)
    run(["go", "test", "./..."], cwd=project, env=env)
    run(["go", "test", "-race", "./..."], cwd=project, env=env)

    if generated_projects[fixture]["context"]["use_db"] == "yes":
        tools = project / "tools"
        run(["go", "mod", "download"], cwd=tools, env=env)
        _assert_tidy_clean(tools, env)
        run(["go", "mod", "tidy"], cwd=tools, env=env)
        _assert_tidy_clean(tools, env)
        run(["go", "build", "./..."], cwd=tools, env=env)
        run(["go", "vet", "./..."], cwd=tools, env=env)


def _assert_tidy_clean(project, env):
    tidy_diff = run(["go", "mod", "tidy", "-diff"], cwd=project, env=env, check=False)
    if tidy_diff.returncode != 0:
        if "flag provided but not defined" in tidy_diff.stderr:
            pytest.skip("go mod tidy -diff no está disponible en esta versión de Go")
        raise AssertionError(
            f"go mod tidy -diff encontró cambios en {project}:\n"
            f"stdout:\n{tidy_diff.stdout}\nstderr:\n{tidy_diff.stderr}"
        )


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
        "tools/cmd/atlas_loader/main.go": use_db,
        "tools/go.mod": use_db,
        "atlas.hcl": use_db,
        "migrations/.gitkeep": use_db,
        "migrations/20261024000000_users.sql": use_db,
        "migrations/atlas.sum": use_db,
        "configs/client_redis.go": use_redis or use_asynq,
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


ARCHITECTURE_BANNED = (
    "configs.Env",
    "configs.GetDB",
    "configs.GetRedis",
    "configs.GetLogger",
    "configs.GetAsynqClient",
    "MustBootstrap",
    "runtime.Caller",
)


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_project_architecture(generated_projects, fixture):
    """Fase 3: sin estado global, sin singletons y con configuración portable."""
    project = generated_projects[fixture]["path"]

    for package in ("pkg", "configs", "internal", "cmd"):
        for path in (project / package).rglob("*.go"):
            content = path.read_text(encoding="utf-8")
            for banned in ARCHITECTURE_BANNED:
                assert banned not in content, f"{path.relative_to(project)} usa {banned}"

    env_files = sorted((project / "env").glob("*.env"))
    assert env_files, "la plantilla debe generar al menos un fichero env"
    for path in env_files:
        if path.name.startswith("_"):
            continue
        values = _read_env_file(path)
        assert "WORKER_CONCURRENCY" in values, f"{path.name} sin WORKER_CONCURRENCY"
        assert int(values["WORKER_CONCURRENCY"]) > 0, f"{path.name}: WORKER_CONCURRENCY inválida"

    worker_concurrency = int(_read_env_file(project / "env/testing.env")["WORKER_CONCURRENCY"])
    configs = (project / "configs/env.go").read_text(encoding="utf-8")
    assert "WorkerConcurrency int" in configs, "Config no expone WorkerConcurrency"
    env_tests = (project / "configs/env_test.go").read_text(encoding="utf-8")
    assert f"WorkerConcurrency: {worker_concurrency}" in env_tests, (
        "los defaults del test de configuración no reflejan WORKER_CONCURRENCY"
    )

    if generated_projects[fixture]["context"]["use_worker"] == "yes":
        worker_main = (project / "cmd/worker/main.go").read_text(encoding="utf-8")
        assert "Concurrency: cfg.WorkerConcurrency" in worker_main
        assert "srv.Shutdown()" in worker_main

    router_source = (project / "internal/server/router.go").read_text(encoding="utf-8")
    assert "/healthz" in router_source and "/readyz" in router_source


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


def test_root_makefile_help():
    """Fase 4.1: el Makefile raíz documenta sus targets y versiones fijadas."""
    if shutil.which("make") is None:
        pytest.skip("make no está instalado")

    result = run(["make", "help"], cwd=ROOT)
    for expected in ("test-template", "security", "Cookiecutter:", "Fixtures:"):
        assert expected in result.stdout, f"make help no menciona {expected}"


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_makefile_targets(generated_projects, fixture):
    """Fase 4.1/4.2: targets disponibles y DSN parametrizados por stage."""
    if shutil.which("make") is None:
        pytest.skip("make no está instalado")

    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]
    use_db = context["use_db"] == "yes"
    use_worker = context["use_worker"] == "yes" and context["use_asynq"] == "yes"
    use_swagger = context["use_swagger"] == "yes"

    targets = ["fmt", "vet", "lint", "build", "build-all", "run", "test", "test-unit", "test-integration", "coverage"]
    if use_db:
        targets += ["migrate", "dbdiff", "dbapply"]
    if use_worker:
        targets.append("worker")
    if use_swagger:
        targets.append("docs")

    for target in targets:
        result = run(["make", "-n", target], cwd=project, check=False)
        assert result.returncode == 0, (
            f"make -n {target} falló en {fixture}:\n{result.stdout}\n{result.stderr}"
        )
        assert result.stdout.strip(), f"make -n {target} no imprime nada en {fixture}"

    if use_db:
        # 4.2: STAGE cambia el env incluido y el DSN de los tests apunta a la
        # base _test, nunca a la de desarrollo.
        migrate_out = run(["make", "-n", "migrate", "STAGE=testing"], cwd=project).stdout
        assert "STAGE=testing go run ./cmd/migrate" in migrate_out

        integration_out = run(["make", "-n", "test-integration"], cwd=project).stdout
        assert f"/{context['project_slug']}_test" in integration_out, integration_out
        assert "dev:dev@127.0.0.1:5432" in integration_out, integration_out

        dbdiff_out = run(["make", "-n", "dbdiff"], cwd=project).stdout
        assert "--var dev_url=" in dbdiff_out, dbdiff_out


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


def _integration_env(project):
    """Entorno para `go test -tags=integration` contra el compose de la prueba.

    Credenciales y host salen de env/testing.env; la base es la que define
    compose.yaml (el slug del proyecto), porque las migraciones del smoke se
    aplican ahí.
    """
    testing = _read_env_file(project / "env" / "testing.env")
    user = testing["DB_USER"]
    password = testing["DB_PASS"]
    host = testing.get("DB_HOST", "127.0.0.1")
    port = testing.get("DB_PORT", "5432")
    sslmode = testing.get("DB_SSL_MODE", "disable")

    env = os.environ.copy()
    env["STAGE"] = "testing"
    env["TEST_DATABASE_URL"] = (
        f"postgres://{user}:{password}@{host}:{port}/{project.name}?sslmode={sslmode}"
    )
    return env


def _post_json(url, payload, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    request = urllib.request.Request(
        url, data=json.dumps(payload).encode("utf-8"), headers=headers, method="POST"
    )
    with urllib.request.urlopen(request, timeout=10) as response:
        return response.status, json.loads(response.read().decode("utf-8"))


def _get_json(url, token=None):
    headers = {"Authorization": f"Bearer {token}"} if token else {}
    request = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(request, timeout=10) as response:
        return response.status, json.loads(response.read().decode("utf-8"))


def test_full_project_http_smoke(generated_projects):
    if "full" not in generated_projects:
        pytest.skip("FIXTURES no incluye el fixture full")
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

        # Migración inicial fuera del arranque de la API (3.4). Con
        # podman-compose el "run" recrea el proyecto, así que se vuelve a
        # levantar la pila y se comprueba que Redis sigue accesible.
        run(["docker", "compose", "--profile", "migrate", "run", "--rm", "atlas"], cwd=project)
        run(["docker", "compose", "up", "-d"], cwd=project)
        assert _wait_for_port(5432), "Postgres no volvió a abrir el puerto"
        assert _wait_for_port(6379), "Redis no volvió a abrir el puerto"

        # 4.4: la estrategia de tests (0.4) también se ejecuta de verdad.
        # Reutiliza la pila ya levantada y migrada; el paquete de repositorios
        # crea y borra su propia tabla, así que no toca el flujo de la API.
        integration_env = _integration_env(project)
        run(["go", "test", "-tags=integration", "./..."], cwd=project, env=integration_env)

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

        # 3.8: readiness con comprobación real de DB y Redis.
        status, ready = _get_json("http://127.0.0.1:8080/readyz")
        assert status == 200, f"/readyz devolvió {status}: {ready}"
        assert ready["status"] == "ok", ready

        # 3.7: flujo de ejemplo signup -> login -> me con JwtAuth.
        email = "smoke@example.com"
        status, body = _post_json(
            "http://127.0.0.1:8080/signup",
            {
                "firstname": "Smoke",
                "lastname": "Test",
                "email": email,
                "password": "s3cret-password",
            },
        )
        assert status == 201, f"/signup devolvió {status}: {body}"
        assert "password" not in json.dumps(body), f"la respuesta filtra la contraseña: {body}"

        status, body = _post_json(
            "http://127.0.0.1:8080/login",
            {"email": email, "password": "s3cret-password"},
        )
        assert status == 200, f"/login devolvió {status}: {body}"
        token = body["data"]["token"]
        assert token, f"sin token en la respuesta: {body}"

        status, body = _get_json("http://127.0.0.1:8080/me", token=token)
        assert status == 200, f"/me devolvió {status}: {body}"
        assert body["data"]["email"] == email, body

        # 3.5: SIGTERM debe terminar con apagado limpio (código 0).
        server.terminate()
        try:
            server.wait(timeout=20)
        except subprocess.TimeoutExpired:
            server.kill()
            server.wait(timeout=10)
            pytest.fail("el servidor no respondió a SIGTERM")
        assert server.returncode == 0, f"código de salida {server.returncode} tras SIGTERM"
        server = None
    finally:
        if server is not None and server.poll() is None:
            server.terminate()
            try:
                server.wait(timeout=10)
            except subprocess.TimeoutExpired:
                server.kill()
        run(["docker", "compose", "down", "-v"], cwd=project, check=False)


def test_generated_tasks_and_generators(generated_projects, tmp_path):
    if "full" not in generated_projects:
        pytest.skip("FIXTURES no incluye el fixture full")
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


def _read_env_file(path):
    values = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        values[key.strip()] = value.strip().strip('"')
    return values


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_security_hardening(generated_projects, fixture):
    project = generated_projects[fixture]["path"]

    dockerfile = (project / "Dockerfile").read_text(encoding="utf-8")
    assert 'CMD ["./http"]' in dockerfile, "el CMD principal no debe ejecutar migraciones"
    assert "atlas migrate apply" not in dockerfile

    entrypoint = project / "entrypoint.sh"
    source = entrypoint.read_text(encoding="utf-8")
    assert "set -eu" in source
    assert 'exec "$@"' in source
    smoke = run(["sh", str(entrypoint), "echo", "entrypoint-ok"], cwd=project)
    assert smoke.stdout.strip() == "entrypoint-ok"

    data = yaml.safe_load(
        (project / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")
    )
    assert "security" in data["jobs"], "falta el job de seguridad en la CI"
    assert data["permissions"]["security-events"] == "write"
    assert data["jobs"]["security"]["timeout-minutes"]

    assert (project / ".gitleaks.toml").exists(), "falta .gitleaks.toml"

    user = (project / "internal" / "core" / "models" / "user.go").read_text(encoding="utf-8")
    password_line = next(
        (line for line in user.splitlines() if "Password" in line), ""
    )
    assert 'json:"-"' in password_line, "Password no debe serializarse en JSON"

    prod = _read_env_file(project / "env" / "prod.env")
    for key in ("ENABLE_SWAGGER", "ENABLE_ASYNQMON", "ENABLE_PPROF"):
        assert prod[key] == "false", f"{key} debe estar desactivado en prod.env"

    for package in ("pkg", "configs", "internal"):
        for path in (project / package).rglob("*.go"):
            content = path.read_text(encoding="utf-8")
            assert "log.Fatal" not in content, f"{path.relative_to(project)} usa log.Fatal"


@pytest.mark.parametrize("fixture", ["full"])
def test_generated_project_gosec(generated_projects, fixture):
    if fixture not in generated_projects:
        pytest.skip("FIXTURES no incluye el fixture full")
    project = generated_projects[fixture]["path"]
    env = environment()

    if shutil.which("gosec") is None:
        if os.environ.get("REQUIRE_SECURITY") == "1":
            raise AssertionError("REQUIRE_SECURITY=1 pero gosec no está instalado")
        pytest.skip("gosec no está instalado")

    run(["gosec", "-quiet", "./..."], cwd=project, env=env)


@pytest.mark.parametrize("fixture", ["full"])
def test_generated_project_gitleaks(generated_projects, fixture):
    if fixture not in generated_projects:
        pytest.skip("FIXTURES no incluye el fixture full")
    project = generated_projects[fixture]["path"]

    if shutil.which("gitleaks") is None:
        if os.environ.get("REQUIRE_SECURITY") == "1":
            raise AssertionError("REQUIRE_SECURITY=1 pero gitleaks no está instalado")
        pytest.skip("gitleaks no está instalado")

    run(
        ["gitleaks", "detect", "--source", ".", "--no-git", "--config", ".gitleaks.toml"],
        cwd=project,
    )

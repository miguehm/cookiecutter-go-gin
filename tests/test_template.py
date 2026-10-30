import json
import os
import re
import shutil
import socket
import subprocess
import time
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

import pexpect
import pytest
import yaml

ROOT = Path(__file__).resolve().parent.parent
TEMPLATE = ROOT / "{{cookiecutter.project_slug}}"
FIXTURES_DIR = ROOT / "tests" / "fixtures"
COPY_WITHOUT_RENDER = [
    "atlas.hcl",
    "tools/templates/*.j2",
    "env/_template.env",
]
COOKIECUTTER_VERSION = os.environ.get("COOKIECUTTER_VERSION") or re.search(
    r"^COOKIECUTTER_VERSION\s*\?=\s*(\S+)",
    (ROOT / "Makefile").read_text(encoding="utf-8"),
    re.MULTILINE,
).group(1)
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
        "cmd/migrate/main.go": use_db,
        "cmd/migrate/main_test.go": use_db,
        "tools/cmd/atlas_loader/main.go": use_db,
        "tools/go.mod": use_db,
        "tools/go.sum": use_db,
        "tools/fake_db.py": use_db,
        "tools/fake.yaml": use_db,
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
        "LICENSE": True,
        "tools/templates/model.j2": True,
        "tools/templates/repository.j2": True,
        "tools/templates/service.j2": True,
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


def test_copy_without_render_contract():
    config = json.loads((ROOT / "cookiecutter.json").read_text(encoding="utf-8"))
    assert config["_copy_without_render"] == COPY_WITHOUT_RENDER


LICENSE_MARKERS = {
    "MIT": ("MIT License", 'THE SOFTWARE IS PROVIDED "AS IS"'),
    "Apache-2.0": (
        "Apache License",
        "Version 2.0, January 2004",
        "http://www.apache.org/licenses/",
    ),
    "BSD-3-Clause": (
        "BSD 3-Clause License",
        "Neither the name of the copyright holder",
    ),
    "proprietary": ("All rights reserved", "proprietary"),
}


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_license(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]

    license_path = project / "LICENSE"
    assert license_path.is_file(), f"{fixture} no generó LICENSE"

    text = license_path.read_text(encoding="utf-8")
    assert context["author"] in text
    year = datetime.now(timezone.utc).year
    assert str(year) in text or str(year - 1) in text, "LICENSE sin el año actual"
    assert "{{" not in text and "{%" not in text
    assert "Nguyễn Bá Cường" not in text
    assert "2018" not in text

    for marker in LICENSE_MARKERS[context["license"]]:
        assert marker in text, f"{fixture} ({context['license']}) sin {marker!r}"

    if context["license"] == "proprietary":
        assert "MIT License" not in text
        assert "Apache License" not in text
        assert "BSD 3-Clause License" not in text


def test_fixture_license_coverage():
    if set(FIXTURES) != set(ALL_FIXTURES):
        pytest.skip("la cobertura de las cuatro licencias requiere la matriz completa")

    licenses = {
        read_fixture(FIXTURES_DIR / f"{fixture}.yaml")["license"]
        for fixture in FIXTURES
    }
    assert licenses == {"MIT", "Apache-2.0", "BSD-3-Clause", "proprietary"}


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_readme_mentions_license(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]
    readme = (project / "README.md").read_text(encoding="utf-8")

    assert "## Licencia" in readme
    if context["license"] == "proprietary":
        assert "propietario" in readme.lower()
    else:
        assert context["license"] in readme


def test_cookiecutter_context_has_no_unused_version():
    config = json.loads((ROOT / "cookiecutter.json").read_text(encoding="utf-8"))
    assert "version" not in config

    for fixture in ALL_FIXTURES:
        context = read_fixture(FIXTURES_DIR / f"{fixture}.yaml")
        assert "version" not in context, f"{fixture} sigue definiendo version"


def _raw_template_files(project):
    files = [project / "env" / "_template.env"]
    files.extend(sorted((project / "tools" / "templates").glob("*.j2")))
    atlas = project / "atlas.hcl"
    if atlas.exists():
        files.append(atlas)
    return files


@pytest.mark.parametrize("fixture", FIXTURES)
def test_copy_without_render_files_are_byte_identical(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]

    relative_paths = ["env/_template.env"]
    relative_paths.extend(
        f"tools/templates/{path.name}" for path in (TEMPLATE / "tools" / "templates").glob("*.j2")
    )
    if context["use_db"] == "yes":
        relative_paths.append("atlas.hcl")

    for relative in relative_paths:
        assert (project / relative).read_bytes() == (TEMPLATE / relative).read_bytes(), (
            f"{relative} cambió al generar el proyecto"
        )


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_project_has_no_unrendered_cookiecutter_jinja(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    raw = {path.relative_to(project).as_posix() for path in _raw_template_files(project)}

    for path in sorted(project.rglob("*")):
        if not path.is_file():
            continue
        relative = path.relative_to(project).as_posix()
        if relative in raw:
            continue
        data = path.read_bytes()
        if b"\x00" in data:
            continue
        content = data.decode("utf-8", errors="ignore")
        assert "{{ cookiecutter." not in content, f"{relative} conserva Jinja sin renderizar"
        assert "{%" not in content, f"{relative} conserva un bloque Jinja sin renderizar"

    assert "{{ STAGE }}" in (project / "env" / "_template.env").read_text(encoding="utf-8")
    for template in sorted((project / "tools" / "templates").glob("*.j2")):
        assert "{{ model }}" in template.read_text(encoding="utf-8"), template
    atlas = project / "atlas.hcl"
    if atlas.exists():
        assert "{{ sql ." in atlas.read_text(encoding="utf-8")


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
def test_generated_compose_services(generated_projects, fixture):
    """5.1: cada servicio de compose aparece solo si su feature está activa."""
    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]
    use_db = context["use_db"] == "yes"
    use_redis = context["use_redis"] == "yes"
    use_asynq = context["use_asynq"] == "yes"

    compose = project / "compose.yaml"
    if not (use_db or use_redis or use_asynq):
        assert not compose.exists(), "sin DB, Redis ni Asynq no debe generarse compose"
        return

    data = yaml.safe_load(compose.read_text(encoding="utf-8"))
    services = data["services"]

    assert ("postgres" in services) is use_db
    assert ("atlas" in services) is use_db
    assert ("redis" in services) is (use_redis or use_asynq)
    assert ("pgdata" in (data.get("volumes") or {})) is use_db


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_ci_conditional_jobs(generated_projects, fixture):
    """5.1: el job de integración y sus servicios siguen a los flags."""
    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]
    use_db = context["use_db"] == "yes"
    use_redis = context["use_redis"] == "yes"
    use_asynq = context["use_asynq"] == "yes"
    needs_services = use_db or use_redis or use_asynq

    data = yaml.safe_load(
        (project / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")
    )
    jobs = data["jobs"]
    assert "test" in jobs and "security" in jobs

    integration = jobs.get("integration")
    if not needs_services:
        assert integration is None, "sin servicios no debe existir el job integration"
        return

    assert integration is not None, "faltan servicios y no hay job integration"
    services = integration.get("services") or {}
    assert ("postgres" in services) is use_db
    assert ("redis" in services) is (use_redis or use_asynq)

    env = integration.get("env") or {}
    assert ("TEST_DATABASE_URL" in env) is use_db
    assert ("REDIS_HOST" in env) is (use_redis or use_asynq)


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_makefile_conditional_targets(generated_projects, fixture):
    """5.1: los targets condicionales existen solo cuando aplica."""
    if shutil.which("make") is None:
        pytest.skip("make no está instalado")

    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]
    expected = {
        "migrate": context["use_db"] == "yes",
        "dbreset": context["use_db"] == "yes",
        "dbdiff": context["use_db"] == "yes",
        "dbapply": context["use_db"] == "yes",
        "worker": context["use_worker"] == "yes" and context["use_asynq"] == "yes",
        "docs": context["use_swagger"] == "yes",
    }

    for target, enabled in expected.items():
        result = run(["make", "-n", target], cwd=project, check=False)
        if enabled:
            assert result.returncode == 0, (
                f"make -n {target} falló en {fixture}:\n{result.stdout}\n{result.stderr}"
            )
        else:
            assert result.returncode != 0, (
                f"{fixture} publica el target {target} sin la feature activa"
            )


_DANGLING_REFERENCE_FILES = (
    "Makefile",
    "Dockerfile",
    ".air.toml",
    "tasks.py",
    "entrypoint.sh",
)


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_project_has_no_dangling_conditional_references(
    generated_projects, fixture
):
    """5.1: ningún archivo operativo apunta a rutas eliminadas por el hook."""
    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]
    use_db = context["use_db"] == "yes"
    use_redis = context["use_redis"] == "yes"
    use_asynq = context["use_asynq"] == "yes"
    use_worker = context["use_worker"] == "yes" and use_asynq

    removed = []
    if not use_db:
        removed += [
            "cmd/migrate",
            "cmd/devcli",
            "tools/cmd",
            "tools/go.mod",
            "tools/fake_db.py",
            "tools/fake.yaml",
            "atlas.hcl",
            "migrations/",
        ]
    if not use_asynq:
        removed += ["internal/core/asynctasks", "configs/client_asynq"]
    if not (use_redis or use_asynq):
        removed.append("configs/client_redis")
    if not use_worker:
        removed.append("cmd/worker")

    paths = [project / name for name in _DANGLING_REFERENCE_FILES]
    paths.extend(sorted((project / ".github" / "workflows").glob("*.yml")))
    compose = project / "compose.yaml"
    if compose.exists():
        paths.append(compose)

    for path in paths:
        if not path.is_file():
            continue
        text = path.read_text(encoding="utf-8")
        for reference in removed:
            assert reference not in text, (
                f"{path.relative_to(project)} referencia {reference!r}, "
                f"eliminado con los flags de {fixture}"
            )

    if use_db:
        assert (project / "tools" / "templates" / "model.j2").exists()
    else:
        assert not (project / "tools" / "go.mod").exists()


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
    for expected in (
        "test-template",
        "security",
        "cookiecutter-version",
        "Cookiecutter:",
        "Fixtures:",
    ):
        assert expected in result.stdout, f"make help no menciona {expected}"


def test_root_makefile_cookiecutter_version():
    """5.6: el pin de Cookiecutter se consulta desde un único sitio."""
    if shutil.which("make") is None:
        pytest.skip("make no está instalado")

    result = run(["make", "-s", "cookiecutter-version"], cwd=ROOT)
    assert result.stdout.strip() == COOKIECUTTER_VERSION

    for readme in (ROOT / "README.md", TEMPLATE / "README.md"):
        text = readme.read_text(encoding="utf-8")
        assert f"cookiecutter=={COOKIECUTTER_VERSION}" not in text, (
            f"{readme} duplica el pin de Cookiecutter; usa `make -s cookiecutter-version`"
        )


def test_renovate_config():
    """5.7: Renovate cubre Go, Docker, Actions, herramientas y Cookiecutter."""
    config = json.loads((ROOT / "renovate.json").read_text(encoding="utf-8"))

    assert config["$schema"].startswith("https://docs.renovatebot.com/")
    assert "config:recommended" in config["extends"]
    assert config["docker-compose"]["enabled"] is False
    assert config["postUpdateOptions"] == ["gomodTidy"], (
        "las actualizaciones Go deben regenerar go.sum"
    )

    managers = config["customManagers"]
    assert managers, "faltan managers personalizados"

    for manager in managers:
        assert manager["customType"] == "regex"
        assert manager.get("managerFilePatterns"), manager
        assert manager.get("matchStrings"), manager

    expected = {
        "cookiecutter": "pypi",
        "go": "golang-version",
        "github.com/golangci/golangci-lint/v2": "go",
        "golang.org/x/vuln": "go",
        "github.com/securego/gosec/v2": "go",
        "gitleaks/gitleaks": "github-releases",
        "aquasecurity/trivy": "github-releases",
    }
    actual = {
        manager["depNameTemplate"]: manager["datasourceTemplate"]
        for manager in managers
        if "depNameTemplate" in manager
    }
    assert actual == expected

    docker = [m for m in managers if m["datasourceTemplate"] == "docker"]
    assert len(docker) == 1, "falta el manager de las imágenes de compose"
    assert any("(?<depName>" in pattern for pattern in docker[0]["matchStrings"])

    grouped = {rule.get("groupName") for rule in config["packageRules"]}
    assert {"módulos Go", "herramientas de CI", "imágenes Docker", "GitHub Actions"} <= grouped


def _pin(text, variable):
    match = re.search(rf"{variable}\s*(?:\?=|:)\s*\"?([^\s\"]+)", text)
    assert match, f"no se encontró {variable}"
    return match.group(1)


def test_renovate_managers_match_the_pins():
    """5.7: los regex de Renovate reconocen las líneas reales de Makefile y CI."""
    config = json.loads((ROOT / "renovate.json").read_text(encoding="utf-8"))
    makefile = (ROOT / "Makefile").read_text(encoding="utf-8")
    ci = (ROOT / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")
    template_ci = (TEMPLATE / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")

    by_dep = {
        manager["depNameTemplate"]: manager
        for manager in config["customManagers"]
        if "depNameTemplate" in manager
    }

    for variable, dep in (
        ("COOKIECUTTER_VERSION", "cookiecutter"),
        ("GOLANGCI_LINT_VERSION", "github.com/golangci/golangci-lint/v2"),
        ("GOVULNCHECK_VERSION", "golang.org/x/vuln"),
        ("GOSEC_VERSION", "github.com/securego/gosec/v2"),
        ("GITLEAKS_VERSION", "gitleaks/gitleaks"),
        ("TRIVY_VERSION", "aquasecurity/trivy"),
    ):
        version = _pin(makefile, variable)
        assert version == _pin(ci, variable), f"{variable} no coincide entre Makefile y CI"
        assert any(
            re.search(pattern.replace("(?<", "(?P<"), makefile)
            for pattern in by_dep[dep]["matchStrings"]
        ), f"Renovate no reconoce {variable} en el Makefile"
        assert any(
            re.search(pattern.replace("(?<", "(?P<"), ci)
            for pattern in by_dep[dep]["matchStrings"]
        ), f"Renovate no reconoce {variable} en la CI"

        if variable in {
            "GOVULNCHECK_VERSION",
            "GOSEC_VERSION",
            "GITLEAKS_VERSION",
            "TRIVY_VERSION",
        }:
            assert _pin(template_ci, variable) == version, (
                f"{variable} no coincide con la CI del proyecto generado"
            )
            assert any(
                re.search(pattern.replace("(?<", "(?P<"), template_ci)
                for pattern in by_dep[dep]["matchStrings"]
            ), f"Renovate no reconoce {variable} en la CI generada"

    go_version = _pin(ci, "GO_VERSION")
    assert any(
        re.search(pattern.replace("(?<", "(?P<"), f'GO_VERSION: "{go_version}"')
        for pattern in by_dep["go"]["matchStrings"]
    )

    docker = next(m for m in config["customManagers"] if m["datasourceTemplate"] == "docker")
    for image in ("postgres:16-alpine", "redis:7-alpine", "arigaio/atlas:1.3.5-alpine"):
        assert any(
            re.search(pattern.replace("(?<", "(?P<"), f"image: {image}")
            for pattern in docker["matchStrings"]
        ), f"Renovate no reconoce la imagen {image}"


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

    # 5.3: acrónimos y comentarios de exportados.
    run(
        ["uvx", "--from", "invoke", "--with", "jinja2", "inv", "gen-model", "HTTPClient"],
        cwd=project,
    )

    generated = {
        "model": project / "internal/core/models/http_client.go",
        "repository": project / "internal/core/repositories/http_client_repository.go",
        "service": project / "internal/core/services/http_client_service.go",
    }
    model_source = generated["model"].read_text(encoding="utf-8")
    assert "// HTTPClient is a persisted entity." in model_source
    assert "type HTTPClient struct" in model_source

    repository_source = generated["repository"].read_text(encoding="utf-8")
    assert "// IHTTPClientRepository is the persistence contract" in repository_source
    assert "func NewHTTPClientRepository(db *gorm.DB) IHTTPClientRepository" in repository_source
    assert "type httpClientRepository struct" in repository_source

    service_source = generated["service"].read_text(encoding="utf-8")
    assert "// HTTPClientService implements" in service_source
    assert "func NewHTTPClientService(repo repositories.IHTTPClientRepository)" in service_source
    assert "// GetAll returns all HTTPClient records." in service_source
    assert "func (s *HTTPClientService) GetAll()" in service_source

    gofmt = shutil.which("gofmt")
    if gofmt is not None:
        for path in generated.values():
            assert run([gofmt, "-l", path], cwd=project).stdout.strip() == ""

    invalid = run(
        ["uvx", "--from", "invoke", "--with", "jinja2", "inv", "gen-model", "http_client"],
        cwd=project,
        check=False,
    )
    assert invalid.returncode != 0, "gen-model debía rechazar un nombre no PascalCase"
    assert "PascalCase" in invalid.stdout + invalid.stderr

    env = environment()
    run(["go", "mod", "tidy"], cwd=project, env=env)
    run(["go", "build", "./..."], cwd=project, env=env)
    run(["go", "test", "./..."], cwd=project, env=env)
    if shutil.which("golangci-lint") is not None:
        run(["golangci-lint", "run", "./..."], cwd=project, env=env)


def test_interactive_generation(tmp_path):
    """5.8: la CLI interactiva pregunta cada variable y genera un proyecto válido."""
    output_dir = tmp_path / "interactive"
    output_dir.mkdir()

    child = pexpect.spawn(
        "uvx",
        [
            "--from",
            f"cookiecutter=={COOKIECUTTER_VERSION}",
            "cookiecutter",
            "--output-dir",
            str(output_dir),
            str(ROOT),
        ],
        encoding="utf-8",
        timeout=180,
    )

    try:
        for prompt, answer in (
            ("project_name", "Interactive Project"),
            ("project_slug", ""),
            ("description", "interactive fixture"),
            ("author", "Example"),
            ("email", "example@example.com"),
            ("go_module", ""),
        ):
            child.expect(prompt)
            child.sendline(answer)

        child.expect("Select license")
        child.expect("Choose from")
        child.sendline("2")

        for flag in ("use_db", "use_redis", "use_asynq"):
            child.expect(flag)
            child.sendline("")

        for flag in ("use_worker", "use_swagger", "use_pprof", "use_sentry"):
            child.expect(flag)
            child.sendline("2")

        child.expect(pexpect.EOF)
    finally:
        child.close()

    transcript = child.before or ""
    assert child.exitstatus == 0, (
        f"cookiecutter interactivo terminó con {child.exitstatus}\n{transcript[-4000:]}"
    )

    project = output_dir / "interactiveproject"
    assert project.is_dir(), f"no se generó {project}"

    license_text = (project / "LICENSE").read_text(encoding="utf-8")
    assert "Apache License" in license_text
    assert "Example" in license_text

    assert not (project / "cmd" / "worker").exists(), "use_worker=no no debe dejar cmd/worker"
    router = (project / "internal" / "server" / "router.go").read_text(encoding="utf-8")
    assert "ginSwagger" not in router
    assert "pprof" not in router

    readme = (project / "README.md").read_text(encoding="utf-8")
    assert "{{" not in readme and "{%" not in readme

    env = environment()
    run(["go", "build", "./..."], cwd=project, env=env)
    run(["go", "vet", "./..."], cwd=project, env=env)
    run(["go", "test", "./..."], cwd=project, env=env)


def test_no_input_generation_without_config(tmp_path):
    """5.8: --no-input sin config usa los defaults de cookiecutter.json."""
    output_dir = tmp_path / "defaults"
    output_dir.mkdir()

    run(
        [
            "uvx",
            "--from",
            f"cookiecutter=={COOKIECUTTER_VERSION}",
            "cookiecutter",
            "--no-input",
            "--output-dir",
            str(output_dir),
            str(ROOT),
        ]
    )

    project = output_dir / "projectname"
    assert project.is_dir(), "los defaults deben generar projectname"

    go_mod = (project / "go.mod").read_text(encoding="utf-8")
    assert "module github.com/yourname/projectname" in go_mod

    license_text = (project / "LICENSE").read_text(encoding="utf-8")
    assert "MIT License" in license_text
    assert "Your Name" in license_text

    assert (project / "cmd" / "worker" / "main.go").exists()
    assert (project / "compose.yaml").exists()
    assert (project / "configs" / "client_db.go").exists()
    assert (project / "configs" / "client_asynq.go").exists()

    readme = (project / "README.md").read_text(encoding="utf-8")
    assert "{{" not in readme and "{%" not in readme

    env = environment()
    run(["go", "build", "./..."], cwd=project, env=env)
    run(["go", "vet", "./..."], cwd=project, env=env)
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

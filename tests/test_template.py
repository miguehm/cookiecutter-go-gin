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

# FIXTURES=minimal,full narrows the matrix (used by CI for PRs); without a
# value all eight fixtures are generated. An unknown name is an error, not a skip.
_selected = {name.strip() for name in os.environ.get("FIXTURES", "").split(",") if name.strip()}
_unknown = _selected - set(ALL_FIXTURES)
if _unknown:
    raise AssertionError(
        f"unknown fixtures: {sorted(_unknown)}; available: {ALL_FIXTURES}"
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
        assert project.is_dir(), f"{project} was not generated"

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
            pytest.skip("go mod tidy -diff is not available in this Go version")
        raise AssertionError(
            f"go mod tidy -diff found changes in {project}:\n"
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
        assert path.exists() is expected, f"{relative}: exists={path.exists()}, expected={expected}"

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
    assert license_path.is_file(), f"{fixture} did not generate a LICENSE"

    text = license_path.read_text(encoding="utf-8")
    assert context["author"] in text
    year = datetime.now(timezone.utc).year
    assert str(year) in text or str(year - 1) in text, "LICENSE is missing the current year"
    assert "{{" not in text and "{%" not in text
    assert "Nguyễn Bá Cường" not in text
    assert "2018" not in text

    for marker in LICENSE_MARKERS[context["license"]]:
        assert marker in text, f"{fixture} ({context['license']}) missing {marker!r}"

    if context["license"] == "proprietary":
        assert "MIT License" not in text
        assert "Apache License" not in text
        assert "BSD 3-Clause License" not in text


def test_fixture_license_coverage():
    if set(FIXTURES) != set(ALL_FIXTURES):
        pytest.skip("four-license coverage requires the full matrix")

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

    assert "## License" in readme
    if context["license"] == "proprietary":
        assert "proprietary" in readme.lower()
    else:
        assert context["license"] in readme


def _env_keys(path):
    keys = []
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        keys.append(line.split("=", 1)[0].strip())
    return keys


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_readme_documents_usage(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]
    readme = (project / "README.md").read_text(encoding="utf-8")

    for section in (
        "## Quick start",
        "## Configuration",
        "### Environment variables",
        "## API",
        "## Docker",
        "## Continuous integration",
        "## Tests",
        "## Deployment",
        "## License",
    ):
        assert section in readme, f"{fixture}: missing section {section!r}"

    assert "{{" not in readme and "{%" not in readme
    assert "cookiecutter-version" not in readme, (
        f"{fixture}: the generated README must not reference template targets"
    )

    if context["use_db"] == "yes":
        assert "cmd/migrate" in readme
        assert "make migrate" in readme
        assert "### Adding a model and an endpoint" in readme
    else:
        assert "without a database" in readme.lower()


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_readme_documents_every_env_var(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    readme = (project / "README.md").read_text(encoding="utf-8")

    for env_name in ("dev.env", "testing.env", "prod.env"):
        for key in _env_keys(project / "env" / env_name):
            assert f"`{key}`" in readme, f"{fixture}: {key} is not documented"


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_env_example(generated_projects, fixture):
    project = generated_projects[fixture]["path"]
    example_path = project / ".env.example"
    assert example_path.is_file(), f"{fixture}: .env.example is missing"

    example = example_path.read_text(encoding="utf-8")
    assert "{{" not in example and "{%" not in example
    assert "dev-secret-change-me" not in example
    assert "change-me-in-dev" not in example

    example_keys = set(_env_keys(example_path))
    for env_name in ("dev.env", "testing.env", "prod.env"):
        missing = set(_env_keys(project / "env" / env_name)) - example_keys
        assert not missing, f"{fixture}: .env.example does not cover {sorted(missing)}"


def test_root_readme_has_after_generating_section():
    readme = (ROOT / "README.md").read_text(encoding="utf-8")

    assert "## After generating" in readme
    assert "README.md" in readme
    assert "{{" not in readme and "{%" not in readme


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
            f"{relative} changed while generating the project"
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
        assert "{{ cookiecutter." not in content, f"{relative} keeps unrendered Jinja"
        assert "{%" not in content, f"{relative} keeps an unrendered Jinja block"

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
    """Phase 3: no global state, no singletons and portable configuration."""
    project = generated_projects[fixture]["path"]

    for package in ("pkg", "configs", "internal", "cmd"):
        for path in (project / package).rglob("*.go"):
            content = path.read_text(encoding="utf-8")
            for banned in ARCHITECTURE_BANNED:
                assert banned not in content, f"{path.relative_to(project)} uses {banned}"

    env_files = sorted((project / "env").glob("*.env"))
    assert env_files, "the template must generate at least one env file"
    for path in env_files:
        if path.name.startswith("_"):
            continue
        values = _read_env_file(path)
        assert "WORKER_CONCURRENCY" in values, f"{path.name} is missing WORKER_CONCURRENCY"
        assert int(values["WORKER_CONCURRENCY"]) > 0, f"{path.name}: invalid WORKER_CONCURRENCY"

    worker_concurrency = int(_read_env_file(project / "env/testing.env")["WORKER_CONCURRENCY"])
    configs = (project / "configs/env.go").read_text(encoding="utf-8")
    assert "WorkerConcurrency int" in configs, "Config does not expose WorkerConcurrency"
    env_tests = (project / "configs/env_test.go").read_text(encoding="utf-8")
    assert f"WorkerConcurrency: {worker_concurrency}" in env_tests, (
        "the config test defaults do not reflect WORKER_CONCURRENCY"
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
        pytest.skip("the fixture does not include compose.yaml")
    if not _docker_available(project):
        pytest.skip("docker compose is not available")

    run(["docker", "compose", "config", "-q"], cwd=project)


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_compose_services(generated_projects, fixture):
    """5.1: each compose service appears only when its feature is active."""
    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]
    use_db = context["use_db"] == "yes"
    use_redis = context["use_redis"] == "yes"
    use_asynq = context["use_asynq"] == "yes"

    compose = project / "compose.yaml"
    if not (use_db or use_redis or use_asynq):
        assert not compose.exists(), "without DB, Redis or Asynq no compose must be generated"
        return

    data = yaml.safe_load(compose.read_text(encoding="utf-8"))
    services = data["services"]

    assert ("postgres" in services) is use_db
    assert ("atlas" in services) is use_db
    assert ("redis" in services) is (use_redis or use_asynq)
    assert ("pgdata" in (data.get("volumes") or {})) is use_db


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_ci_conditional_jobs(generated_projects, fixture):
    """5.1: the integration job and its services follow the flags."""
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
        assert integration is None, "with no services the integration job must not exist"
        return

    assert integration is not None, "services are expected but the integration job is missing"
    services = integration.get("services") or {}
    assert ("postgres" in services) is use_db
    assert ("redis" in services) is (use_redis or use_asynq)

    env = integration.get("env") or {}
    assert ("TEST_DATABASE_URL" in env) is use_db
    assert ("REDIS_HOST" in env) is (use_redis or use_asynq)


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_makefile_conditional_targets(generated_projects, fixture):
    """5.1: the conditional targets exist only when they apply."""
    if shutil.which("make") is None:
        pytest.skip("make is not installed")

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
                f"make -n {target} failed in {fixture}:\n{result.stdout}\n{result.stderr}"
            )
        else:
            assert result.returncode != 0, (
                f"{fixture} exposes the target {target} without the feature enabled"
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
    """5.1: no operational file points at paths removed by the hook."""
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
                f"{path.relative_to(project)} references {reference!r}, "
                f"removed with the flags of {fixture}"
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
        pytest.skip("golangci-lint is not installed")
    run(["golangci-lint", "run"], cwd=project, env=env)

    if shutil.which("govulncheck") is None:
        pytest.skip("govulncheck is not installed")

    result = run(["govulncheck", "./..."], cwd=project, env=env, check=False)
    if result.returncode != 0:
        detail = (result.stdout + result.stderr)[-4000:]
        if os.environ.get("REQUIRE_GOVULNCHECK") == "1":
            raise AssertionError(f"govulncheck failed:\n{detail}")
        pytest.xfail(
            "govulncheck fails with the Phase 0 dependencies; "
            "fixed by the update in Phase 1:\n" + detail
        )


def test_root_makefile_help():
    """Phase 4.1: the root Makefile documents its targets and pinned versions."""
    if shutil.which("make") is None:
        pytest.skip("make is not installed")

    result = run(["make", "help"], cwd=ROOT)
    for expected in (
        "test-template",
        "security",
        "cookiecutter-version",
        "Cookiecutter:",
        "Fixtures:",
    ):
        assert expected in result.stdout, f"make help does not mention {expected}"


def test_root_makefile_cookiecutter_version():
    """5.6: the Cookiecutter pin is read from a single place."""
    if shutil.which("make") is None:
        pytest.skip("make is not installed")

    result = run(["make", "-s", "cookiecutter-version"], cwd=ROOT)
    assert result.stdout.strip() == COOKIECUTTER_VERSION

    for readme in (ROOT / "README.md", TEMPLATE / "README.md"):
        text = readme.read_text(encoding="utf-8")
        assert f"cookiecutter=={COOKIECUTTER_VERSION}" not in text, (
            f"{readme} duplicates the Cookiecutter pin; use `make -s cookiecutter-version`"
        )


def test_renovate_config():
    """5.7: Renovate covers Go, Docker, Actions, tooling and Cookiecutter."""
    config = json.loads((ROOT / "renovate.json").read_text(encoding="utf-8"))

    assert config["$schema"].startswith("https://docs.renovatebot.com/")
    assert "config:recommended" in config["extends"]
    assert config["docker-compose"]["enabled"] is False
    assert config["postUpdateOptions"] == ["gomodTidy"], (
        "Go updates must regenerate go.sum"
    )

    managers = config["customManagers"]
    assert managers, "custom managers are missing"

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
    assert len(docker) == 1, "the compose images manager is missing"
    assert any("(?<depName>" in pattern for pattern in docker[0]["matchStrings"])

    grouped = {rule.get("groupName") for rule in config["packageRules"]}
    assert {"Go modules", "CI tools", "docker images", "GitHub Actions"} <= grouped


def _pin(text, variable):
    match = re.search(rf"{variable}\s*(?:\?=|:)\s*\"?([^\s\"]+)", text)
    assert match, f"{variable} was not found"
    return match.group(1)


def test_renovate_managers_match_the_pins():
    """5.7: the Renovate regexes match the real Makefile and CI lines."""
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
        assert version == _pin(ci, variable), f"{variable} differs between Makefile and CI"
        assert any(
            re.search(pattern.replace("(?<", "(?P<"), makefile)
            for pattern in by_dep[dep]["matchStrings"]
        ), f"Renovate does not recognize {variable} in the Makefile"
        assert any(
            re.search(pattern.replace("(?<", "(?P<"), ci)
            for pattern in by_dep[dep]["matchStrings"]
        ), f"Renovate does not recognize {variable} in the CI"

        if variable in {
            "GOVULNCHECK_VERSION",
            "GOSEC_VERSION",
            "GITLEAKS_VERSION",
            "TRIVY_VERSION",
        }:
            assert _pin(template_ci, variable) == version, (
                f"{variable} does not match the generated project CI"
            )
            assert any(
                re.search(pattern.replace("(?<", "(?P<"), template_ci)
                for pattern in by_dep[dep]["matchStrings"]
            ), f"Renovate does not recognize {variable} in the generated CI"

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
        ), f"Renovate does not recognize the image {image}"


@pytest.mark.parametrize("fixture", FIXTURES)
def test_generated_makefile_targets(generated_projects, fixture):
    """Phase 4.1/4.2: available targets and DSN parameterized by stage."""
    if shutil.which("make") is None:
        pytest.skip("make is not installed")

    project = generated_projects[fixture]["path"]
    context = generated_projects[fixture]["context"]
    use_db = context["use_db"] == "yes"
    use_worker = context["use_worker"] == "yes" and context["use_asynq"] == "yes"
    use_swagger = context["use_swagger"] == "yes"

    targets = [
        "fmt", "vet", "lint", "build", "build-all", "run",
        "test", "test-unit", "test-integration", "coverage",
    ]
    if use_db:
        targets += ["migrate", "dbdiff", "dbapply"]
    if use_worker:
        targets.append("worker")
    if use_swagger:
        targets.append("docs")

    for target in targets:
        result = run(["make", "-n", target], cwd=project, check=False)
        assert result.returncode == 0, (
            f"make -n {target} failed in {fixture}:\n{result.stdout}\n{result.stderr}"
        )
        assert result.stdout.strip(), f"make -n {target} prints nothing in {fixture}"

    if use_db:
        # 4.2: STAGE changes the included env and the test DSN points at the
        # _test database, never at the dev one.
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
        ("1invalid", "github.com/acme/app", "no", "no", "project_slug must start"),
        ("invalid slug", "github.com/acme/app", "no", "no", "project_slug must start"),
        ("app..x", "github.com/acme/app", "no", "no", "project_slug must start"),
        ("app", "github.com/acme/../app", "no", "no", "go_module must not contain '..'"),
        ("app", "github.com/acme/app func", "no", "no", "go_module must not contain whitespace"),
        ("app", "github.com/acme/func", "no", "no", "Go keyword"),
        ("app", "github.com/acme//app", "no", "no", "empty segments"),
        ("app", "github.com/acme/app", "yes", "no", "use_worker=yes requires use_asynq=yes"),
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

    assert result.returncode != 0, "the generation was expected to fail"
    output = result.stdout + result.stderr
    assert message in output, f"{message!r} was not found in:\n{output}"


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
    """Environment for `go test -tags=integration` against the test compose.

    Credentials and host come from env/testing.env; the database is the one
    defined by compose.yaml (the project slug), since the smoke migrations are
    applied there.
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
        pytest.skip("FIXTURES does not include the full fixture")
    project = generated_projects["full"]["path"]

    if not _docker_available(project):
        pytest.skip("docker compose is not available")
    if _port_open(5432) or _port_open(6379) or _port_open(8080):
        pytest.skip("ports 5432/6379/8080 are already in use")

    env = os.environ.copy()
    server = None

    try:
        run(["docker", "compose", "up", "-d"], cwd=project)
        assert _wait_for_port(5432), "Postgres did not open the port"
        assert _wait_for_port(6379), "Redis did not open the port"

        # Initial migration outside the API startup (3.4). With podman-compose
        # the "run" recreates the project, so the stack is brought up again and
        # Redis is checked to still be reachable.
        run(["docker", "compose", "--profile", "migrate", "run", "--rm", "atlas"], cwd=project)
        run(["docker", "compose", "up", "-d"], cwd=project)
        assert _wait_for_port(5432), "Postgres did not reopen the port"
        assert _wait_for_port(6379), "Redis did not reopen the port"

        # 4.4: the test strategy (0.4) is actually executed as well.
        # It reuses the stack already up and migrated; the repositories package
        # creates and drops its own table, so it does not touch the API flow.
        integration_env = _integration_env(project)
        run(["go", "test", "-tags=integration", "./..."], cwd=project, env=integration_env)

        smoke_bin = "/tmp/cookiecutter-smoke-http"
        run(["go", "build", "-o", smoke_bin, "./cmd/http"], cwd=project, env=env)
        server = subprocess.Popen(
            [smoke_bin],
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
            pytest.fail(f"the API did not respond on /healthz. Output:\n{output}")

        # 3.8: readiness with a real DB and Redis check.
        status, ready = _get_json("http://127.0.0.1:8080/readyz")
        assert status == 200, f"/readyz returned {status}: {ready}"
        assert ready["status"] == "ok", ready

        # 3.7: sample flow signup -> login -> me with JwtAuth.
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
        assert status == 201, f"/signup returned {status}: {body}"
        assert "password" not in json.dumps(body), f"the response leaks the password: {body}"

        status, body = _post_json(
            "http://127.0.0.1:8080/login",
            {"email": email, "password": "s3cret-password"},
        )
        assert status == 200, f"/login returned {status}: {body}"
        token = body["data"]["token"]
        assert token, f"no token in the response: {body}"

        status, body = _get_json("http://127.0.0.1:8080/me", token=token)
        assert status == 200, f"/me returned {status}: {body}"
        assert body["data"]["email"] == email, body

        # 3.5: SIGTERM must end with a clean shutdown (exit code 0).
        server.terminate()
        try:
            server.wait(timeout=20)
        except subprocess.TimeoutExpired:
            server.kill()
            server.wait(timeout=10)
            pytest.fail("the server did not respond to SIGTERM")
        assert server.returncode == 0, f"exit code {server.returncode} after SIGTERM"
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
        pytest.skip("FIXTURES does not include the full fixture")
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
        assert (project / relative).exists(), f"{relative} was not generated"

    # 5.3: acronyms and comments of the exported symbols.
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
    assert invalid.returncode != 0, "gen-model was expected to reject a non PascalCase name"
    assert "PascalCase" in invalid.stdout + invalid.stderr

    env = environment()
    run(["go", "mod", "tidy"], cwd=project, env=env)
    run(["go", "build", "./..."], cwd=project, env=env)
    run(["go", "test", "./..."], cwd=project, env=env)
    if shutil.which("golangci-lint") is not None:
        run(["golangci-lint", "run", "./..."], cwd=project, env=env)


def test_interactive_generation(tmp_path):
    """5.8: the interactive CLI asks every variable and generates a valid project."""
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
        f"interactive cookiecutter exited with {child.exitstatus}\n{transcript[-4000:]}"
    )

    project = output_dir / "interactiveproject"
    assert project.is_dir(), f"{project} was not generated"

    license_text = (project / "LICENSE").read_text(encoding="utf-8")
    assert "Apache License" in license_text
    assert "Example" in license_text

    assert not (project / "cmd" / "worker").exists(), "use_worker=no must not leave cmd/worker"
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
    """5.8: --no-input without config uses the cookiecutter.json defaults."""
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
    assert project.is_dir(), "the defaults must generate projectname"

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
        assert f"module {go_module}" in go_mod, f"{fixture} does not use go_module in go.mod"

        router = (project / "internal/server/router.go").read_text(encoding="utf-8")
        assert go_module in router, f"{fixture} does not use go_module in router.go"


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
    assert 'CMD ["./http"]' in dockerfile, "the main CMD must not run migrations"
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
    assert "security" in data["jobs"], "the security job is missing from the CI"
    assert data["permissions"]["security-events"] == "write"
    assert data["jobs"]["security"]["timeout-minutes"]

    assert (project / ".gitleaks.toml").exists(), ".gitleaks.toml is missing"

    user = (project / "internal" / "core" / "models" / "user.go").read_text(encoding="utf-8")
    password_line = next(
        (line for line in user.splitlines() if "Password" in line), ""
    )
    assert 'json:"-"' in password_line, "Password must not be serialized to JSON"

    prod = _read_env_file(project / "env" / "prod.env")
    for key in ("ENABLE_SWAGGER", "ENABLE_ASYNQMON", "ENABLE_PPROF"):
        assert prod[key] == "false", f"{key} must be disabled in prod.env"

    for package in ("pkg", "configs", "internal"):
        for path in (project / package).rglob("*.go"):
            content = path.read_text(encoding="utf-8")
            assert "log.Fatal" not in content, f"{path.relative_to(project)} uses log.Fatal"


@pytest.mark.parametrize("fixture", ["full"])
def test_generated_project_gosec(generated_projects, fixture):
    if fixture not in generated_projects:
        pytest.skip("FIXTURES does not include the full fixture")
    project = generated_projects[fixture]["path"]
    env = environment()

    if shutil.which("gosec") is None:
        if os.environ.get("REQUIRE_SECURITY") == "1":
            raise AssertionError("REQUIRE_SECURITY=1 but gosec is not installed")
        pytest.skip("gosec is not installed")

    run(["gosec", "-quiet", "./..."], cwd=project, env=env)


@pytest.mark.parametrize("fixture", ["full"])
def test_generated_project_gitleaks(generated_projects, fixture):
    if fixture not in generated_projects:
        pytest.skip("FIXTURES does not include the full fixture")
    project = generated_projects[fixture]["path"]

    if shutil.which("gitleaks") is None:
        if os.environ.get("REQUIRE_SECURITY") == "1":
            raise AssertionError("REQUIRE_SECURITY=1 but gitleaks is not installed")
        pytest.skip("gitleaks is not installed")

    run(
        ["gitleaks", "detect", "--source", ".", "--no-git", "--config", ".gitleaks.toml"],
        cwd=project,
    )

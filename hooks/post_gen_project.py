import shutil
import subprocess
import sys
from pathlib import Path

PROJECT_ROOT = Path.cwd()
PROJECT_SLUG = "{{ cookiecutter.project_slug }}"


def remove(path):
    target = PROJECT_ROOT / path
    if target.is_dir():
        shutil.rmtree(target)
    elif target.exists():
        target.unlink()


def fail(message):
    print(f"ERROR: {message}", file=sys.stderr)
    sys.exit(1)


USE_DB = {{ "True" if cookiecutter.use_db == "yes" else "False" }}
USE_REDIS = {{ "True" if cookiecutter.use_redis == "yes" else "False" }}
USE_ASYNQ = {{ "True" if cookiecutter.use_asynq == "yes" else "False" }}
USE_WORKER = {{ "True" if cookiecutter.use_worker == "yes" else "False" }}

REMOVALS = [
    ("configs/client_db.go", not USE_DB),
    ("cmd/devcli", not USE_DB),
    ("cmd/migrate", not USE_DB),
    ("tools/cmd", not USE_DB),
    ("tools/go.mod", not USE_DB),
    ("tools/go.sum", not USE_DB),
    ("atlas.hcl", not USE_DB),
    ("migrations", not USE_DB),
    ("tools/fake_db.py", not USE_DB),
    ("tools/fake.yaml", not USE_DB),
    ("configs/client_redis.go", not USE_REDIS and not USE_ASYNQ),
    ("configs/client_asynq.go", not USE_ASYNQ),
    ("internal/core/asynctasks", not USE_ASYNQ),
    ("cmd/worker", not (USE_WORKER and USE_ASYNQ)),
    ("compose.yaml", not (USE_DB or USE_REDIS or USE_ASYNQ)),
]

for path, should_remove in REMOVALS:
    if should_remove:
        remove(path)

if not (PROJECT_ROOT / "go.mod").exists():
    fail("go.mod was not generated")

gofmt = shutil.which("gofmt")
if gofmt is None:
    print("WARNING: gofmt is not installed; Go files have not been formatted", file=sys.stderr)
else:
    go_files = [str(path) for path in PROJECT_ROOT.rglob("*.go")]
    if go_files:
        result = subprocess.run(
            [gofmt, "-w", *go_files],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            fail(f"gofmt failed: {result.stderr.strip()}")

print(f"Project generated at {PROJECT_ROOT}")
print("Next steps:")
print(f"  cd {PROJECT_SLUG}")
if USE_DB or USE_REDIS or USE_ASYNQ:
    print("  docker compose up -d")
print("  go mod tidy")
print("  go build ./...")
print("  go test ./...")
print("  make run")

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


{% if cookiecutter.use_db == "no" %}
for path in [
    "configs/client_db.go",
    "tools/cmd/atlas_loader",
    "tools/cmd",
    "tools/go.mod",
    "tools/go.sum",
    "atlas.hcl",
    "migrations",
    "tools/fake_db.py",
    "tools/fake.yaml",
]:
    remove(path)
{% endif %}

{% if cookiecutter.use_db == "no" %}
remove("cmd/devcli")
{% endif %}

{% if cookiecutter.use_redis == "no" %}
remove("configs/client_redis.go")
{% endif %}

{% if cookiecutter.use_asynq == "no" %}
for path in ["configs/client_asynq.go", "internal/core/asynctasks"]:
    remove(path)
{% endif %}

{% if cookiecutter.use_worker == "no" or cookiecutter.use_asynq == "no" %}
remove("cmd/worker")
{% endif %}

{% if cookiecutter.use_db == "no" and cookiecutter.use_redis == "no" and cookiecutter.use_asynq == "no" %}
remove("compose.yaml")
{% endif %}

if not (PROJECT_ROOT / "go.mod").exists():
    fail("no se generó go.mod")

gofmt = shutil.which("gofmt")
if gofmt is None:
    print("WARNING: gofmt no está instalado; no se han formateado los archivos Go", file=sys.stderr)
else:
    go_files = [str(path) for path in PROJECT_ROOT.rglob("*.go")]
    if go_files:
        result = subprocess.run(
            [gofmt, "-w", *go_files],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            fail(f"gofmt falló: {result.stderr.strip()}")

print(f"Proyecto generado en {PROJECT_ROOT}")
print("Siguientes pasos:")
print(f"  cd {PROJECT_SLUG}")
{% if cookiecutter.use_db == "yes" or cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}
print("  docker compose up -d")
{% endif %}
print("  go mod tidy")
print("  go build ./...")
print("  go test ./...")
print("  make run")

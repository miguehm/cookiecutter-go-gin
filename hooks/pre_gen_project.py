import re
import sys

project_slug = {{ cookiecutter.project_slug | tojson }}
go_module = {{ cookiecutter.go_module | tojson }}
use_asynq = {{ cookiecutter.use_asynq | tojson }}
use_worker = {{ cookiecutter.use_worker | tojson }}

GO_KEYWORDS = {
    "break",
    "default",
    "func",
    "interface",
    "select",
    "case",
    "defer",
    "go",
    "map",
    "struct",
    "chan",
    "else",
    "goto",
    "package",
    "switch",
    "const",
    "fallthrough",
    "if",
    "range",
    "type",
    "continue",
    "for",
    "import",
    "return",
    "var",
}


def fail(message):
    print(f"ERROR: {message}", file=sys.stderr)
    sys.exit(1)


def validate_slug(slug):
    if not slug:
        fail("project_slug no puede estar vacío")

    if not re.fullmatch(r"[a-z][a-z0-9_-]*", slug):
        fail(
            "project_slug debe empezar por una letra minúscula y contener solo "
            "letras minúsculas, dígitos, '_' o '-'"
        )

    if slug in GO_KEYWORDS:
        fail(f"project_slug no puede ser una palabra reservada de Go: {slug}")


def validate_module(module):
    if not module:
        fail("go_module no puede estar vacío")

    if any(character.isspace() for character in module):
        fail("go_module no puede contener espacios")

    if ".." in module:
        fail("go_module no puede contener '..'")

    parts = module.split("/")
    if any(part == "" for part in parts):
        fail("go_module no puede contener segmentos vacíos")

    for part in parts:
        if part in (".", ".."):
            fail(f"go_module contiene un segmento inválido: {part}")

        if not re.fullmatch(r"[A-Za-z0-9._~-]+", part):
            fail(f"go_module contiene caracteres inválidos: {part}")

        if part[0] in ".-":
            fail(f"los segmentos de go_module no pueden empezar por '.' o '-': {part}")

        if part in GO_KEYWORDS:
            fail(f"go_module no puede usar una palabra reservada de Go: {part}")


validate_slug(project_slug)
validate_module(go_module)

if use_worker == "yes" and use_asynq == "no":
    fail("use_worker=yes requiere use_asynq=yes")

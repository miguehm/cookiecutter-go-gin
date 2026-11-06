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
        fail("project_slug must not be empty")

    if not re.fullmatch(r"[a-z][a-z0-9_-]*", slug):
        fail(
            "project_slug must start with a lowercase letter and contain only "
            "lowercase letters, digits, '_' or '-'"
        )

    if slug in GO_KEYWORDS:
        fail(f"project_slug must not be a Go keyword: {slug}")


def validate_module(module):
    if not module:
        fail("go_module must not be empty")

    if any(character.isspace() for character in module):
        fail("go_module must not contain whitespace")

    if ".." in module:
        fail("go_module must not contain '..'")

    parts = module.split("/")
    if any(part == "" for part in parts):
        fail("go_module must not contain empty segments")

    for part in parts:
        if part in (".", ".."):
            fail(f"go_module contains an invalid segment: {part}")

        if not re.fullmatch(r"[A-Za-z0-9._~-]+", part):
            fail(f"go_module contains invalid characters: {part}")

        if part[0] in ".-":
            fail(f"go_module segments must not start with '.' or '-': {part}")

        if part in GO_KEYWORDS:
            fail(f"go_module must not use a Go keyword: {part}")


validate_slug(project_slug)
validate_module(go_module)

if use_worker == "yes" and use_asynq == "no":
    fail("use_worker=yes requires use_asynq=yes")

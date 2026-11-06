# !/usr/bin/env python3
"""
uvx --from invoke --with jinja2 inv <task>
"""

import contextlib
import logging
import os
import re
import shutil
import subprocess

import jinja2
from invoke import task
from jinja2 import Environment, FileSystemLoader

# Logging configs
# -------------------------------------------------------------------------
FORMATTER = logging.Formatter('%(asctime)s %(levelname)s %(message)s')

logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)

console_handler = logging.StreamHandler()
console_handler.setFormatter(FORMATTER)
logger.addHandler(console_handler)


# Utils
# -------------------------------------------------------------------------
@contextlib.contextmanager
def on_working_dir(dirname=None):
    cwd = os.getcwd()
    try:
        if dirname is not None:
            os.chdir(dirname)
        yield
    finally:
        os.chdir(cwd)


# Utils
# -------------------------------------------------------------------------
MODEL_NAME = re.compile(r"^[A-Z][A-Za-z0-9]*$")


def to_snake_case(s):
    # Splits acronyms before the lowercase → uppercase transitions:
    # HTTPServer → http_server and HarnessModel → harness_model.
    s = re.sub(r"([A-Z]+)([A-Z][a-z])", r"\1_\2", s)
    s = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", s)
    return s.lower()


def to_camel_case(s):
    words = to_snake_case(s).split("_")
    return words[0] + "".join(word.capitalize() for word in words[1:])


def validate_model_name(model):
    if not MODEL_NAME.match(model):
        logger.error(
            "the model name must be PascalCase and a valid Go identifier "
            "(for example HarnessModel)"
        )
        raise SystemExit(1)


def read_go_module():
    with open('go.mod') as file:
        for line in file:
            if line.startswith('module '):
                return line.split(None, 1)[1].strip()

    raise RuntimeError("the 'module' line was not found in go.mod")


def template_context(model):
    validate_model_name(model)
    return {
        'model': model,
        'model_camelcase': to_camel_case(model),
        'go_module': read_go_module(),
    }


def gofmt(path):
    gofmt_bin = shutil.which("gofmt")
    if gofmt_bin is None:
        logger.warning("gofmt is not installed; %s was not formatted", path)
        return

    result = subprocess.run([gofmt_bin, "-w", path], capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"gofmt failed in {path}: {result.stderr.strip()}")


# Tasks
# -------------------------------------------------------------------------
def read_config_file(file_path):
    config = {}

    with open(file_path) as file:
        for line in file:
            line = line.strip()

            if not line or line.startswith("#"):
                continue

            key, value = line.split("=", 1)
            config[key.strip()] = value.strip()

    return config


def _format_env(stage):
    logger.info(f"format env for stage {stage}")
    with open('env/_template.env') as file:
        template_content = file.read()

    config = read_config_file(f'env/{stage}.env')
    template = jinja2.Template(template_content)
    with open(f'env/{stage}.env', 'w') as output_file:
        output_file.write(template.render(config))


@task
def format_env(c, stage="all"):
    if stage == "all":
        for name in sorted(os.listdir('env')):
            if name.endswith('.env') and not name.startswith('_'):
                _format_env(name[:-4])
    else:
        _format_env(stage)


@task
def gen_repository(c, model):
    env = Environment(
        loader=FileSystemLoader(searchpath='./tools/templates'),
        keep_trailing_newline=True,
    )
    template = env.get_template('repository.j2')
    rendered_content = template.render(template_context(model))

    out_file = f'internal/core/repositories/{to_snake_case(model)}_repository.go'
    with open(out_file, 'w') as f:
        f.write(rendered_content)
    gofmt(out_file)

    logger.info(f"generated: {out_file}")


@task
def gen_service(c, model):
    env = Environment(
        loader=FileSystemLoader(searchpath='./tools/templates'),
        keep_trailing_newline=True,
    )
    template = env.get_template('service.j2')
    rendered_content = template.render(template_context(model))

    out_file = f'internal/core/services/{to_snake_case(model)}_service.go'
    with open(out_file, 'w') as f:
        f.write(rendered_content)
    gofmt(out_file)

    logger.info(f"generated: {out_file}")


@task
def gen_model(c, model):
    env = Environment(
        loader=FileSystemLoader(searchpath='./tools/templates'),
        keep_trailing_newline=True,
    )
    template = env.get_template('model.j2')
    rendered_content = template.render(template_context(model))

    out_file = f'internal/core/models/{to_snake_case(model)}.go'
    with open(out_file, 'w') as f:
        f.write(rendered_content)
    gofmt(out_file)

    logger.info(f"generated: {out_file}")

    gen_repository(c, model)
    gen_service(c, model)

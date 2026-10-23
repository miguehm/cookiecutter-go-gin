# !/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
uvx --from invoke --with jinja2 inv <task>
"""

import contextlib
import logging
import os
import re

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
def to_snake_case(s):
    snake_case = re.sub(r'(?<!^)(?=[A-Z])', '_', s).lower()
    return snake_case


def to_camel_case(s):
    return s[0].lower() + s[1:] if s else s


def read_go_module():
    with open('go.mod', 'r') as file:
        for line in file:
            if line.startswith('module '):
                return line.split(None, 1)[1].strip()

    raise RuntimeError("no se encontró la línea 'module' en go.mod")


def template_context(model):
    return {
        'model': model,
        'model_camelcase': to_camel_case(model),
        'go_module': read_go_module(),
    }


# Tasks
# -------------------------------------------------------------------------
def read_config_file(file_path):
    config = {}

    with open(file_path, 'r') as file:
        for line in file:
            line = line.strip()

            if not line or line.startswith("#"):
                continue

            key, value = line.split("=", 1)
            config[key.strip()] = value.strip()

    return config


def _format_env(stage):
    logger.info(f"format env for stage {stage}")
    with open('env/_template.env', 'r') as file:
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
    env = Environment(loader=FileSystemLoader(searchpath='./tools/templates'))
    template = env.get_template('repository.j2')
    rendered_content = template.render(template_context(model))

    out_file = f'internal/core/repositories/{to_snake_case(model)}_repository.go'
    with open(out_file, 'w') as f:
        f.write(rendered_content)

    logger.info(f"generated: {out_file}")


@task
def gen_service(c, model):
    env = Environment(loader=FileSystemLoader(searchpath='./tools/templates'))
    template = env.get_template('service.j2')
    rendered_content = template.render(template_context(model))

    out_file = f'internal/core/services/{to_snake_case(model)}_service.go'
    with open(out_file, 'w') as f:
        f.write(rendered_content)

    logger.info(f"generated: {out_file}")


@task
def gen_model(c, model):
    env = Environment(loader=FileSystemLoader(searchpath='./tools/templates'))
    template = env.get_template('model.j2')
    rendered_content = template.render(template_context(model))

    out_file = f'internal/core/models/{to_snake_case(model)}.go'
    with open(out_file, 'w') as f:
        f.write(rendered_content)

    logger.info(f"generated: {out_file}")

    gen_repository(c, model)
    gen_service(c, model)

COOKIECUTTER_VERSION ?= 2.7.1

.PHONY: help test-template

help:
	@echo "Targets:"
	@echo "  test-template  Genera los fixtures y valida los proyectos resultantes"
	@echo ""
	@echo "Cookiecutter: $(COOKIECUTTER_VERSION)"

test-template:
	COOKIECUTTER_VERSION=$(COOKIECUTTER_VERSION) \
	uvx --with pytest --with pyyaml --with pexpect pytest -q tests/

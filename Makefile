COOKIECUTTER_VERSION ?= 2.7.1
GOLANGCI_LINT_VERSION ?= v2.15.0
GOVULNCHECK_VERSION ?= v1.9.0
GOSEC_VERSION ?= v2.30.0
GITLEAKS_VERSION ?= v8.28.0
TRIVY_VERSION ?= v0.76.1

# Fixtures que genera el arnés; vacío significa los ocho del repositorio.
FIXTURES ?=

.PHONY: help test-template security

help:
	@echo "Targets:"
	@echo "  test-template  Genera los fixtures y valida los proyectos resultantes"
	@echo "                 FIXTURES=minimal,full para acotar la matriz"
	@echo "  security       Escanea el template con gitleaks y trivy (si están disponibles)"
	@echo "                 gosec y govulncheck corren dentro de test-template"
	@echo ""
	@echo "Cookiecutter:    $(COOKIECUTTER_VERSION)"
	@echo "golangci-lint:   $(GOLANGCI_LINT_VERSION)"
	@echo "govulncheck:     $(GOVULNCHECK_VERSION)"
	@echo "gosec:           $(GOSEC_VERSION)"
	@echo "gitleaks:        $(GITLEAKS_VERSION)"
	@echo "trivy:           $(TRIVY_VERSION)"
	@echo "Fixtures:        $(if $(FIXTURES),$(FIXTURES),todos)"

test-template:
	FIXTURES="$(FIXTURES)" \
	COOKIECUTTER_VERSION=$(COOKIECUTTER_VERSION) \
	uvx --with pytest --with pyyaml --with pexpect pytest -q tests/

security:
	@if command -v gitleaks >/dev/null 2>&1; then \
		gitleaks detect --source . --no-git --config .gitleaks.toml; \
	else \
		echo "gitleaks no disponible"; \
	fi
	@if command -v trivy >/dev/null 2>&1; then \
		trivy fs --scanners secret --exit-code 1 --skip-dirs .opencode . && \
		trivy fs --scanners vuln --exit-code 0 --skip-dirs .opencode .; \
	else \
		echo "trivy no disponible"; \
	fi

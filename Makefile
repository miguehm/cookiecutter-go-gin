COOKIECUTTER_VERSION ?= 2.7.1
GOSEC_VERSION ?= v2.30.0
GITLEAKS_VERSION ?= v8.28.0
TRIVY_VERSION ?= v0.76.1

.PHONY: help test-template security

help:
	@echo "Targets:"
	@echo "  test-template  Genera los fixtures y valida los proyectos resultantes"
	@echo "  security       Escanea el template con gitleaks y trivy (si están disponibles)"
	@echo "                 gosec y govulncheck corren dentro de test-template"
	@echo ""
	@echo "Cookiecutter: $(COOKIECUTTER_VERSION)"
	@echo "gosec:        $(GOSEC_VERSION)"
	@echo "gitleaks:     $(GITLEAKS_VERSION)"
	@echo "trivy:        $(TRIVY_VERSION)"

test-template:
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

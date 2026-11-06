.PHONY: fmt vet lint build build-all run test test-unit test-integration coverage{% if cookiecutter.use_db == "yes" %} migrate dbreset dbdiff dbapply{% endif %}{% if cookiecutter.use_swagger == "yes" %} docs{% endif %}{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %} worker{% endif %}

# Stage used by the targets that start the application or the database.
# It inherits STAGE from the environment when already defined; a command line
# (make STAGE=testing run) always wins over the included file.
STAGE ?= dev

# Loads env/<stage>.env as make variables so host, port and credentials live
# in a single place (4.2). `-include` does not fail if the stage does not
# exist: the `?=` below are the fallbacks.
-include env/$(STAGE).env

{% if cookiecutter.use_db == "yes" %}
# Fallbacks in case env/<stage>.env is not present.
DB_HOST ?= 127.0.0.1
DB_PORT ?= 5432
DB_USER ?= dev
DB_PASS ?= dev
DB_NAME ?= {{ cookiecutter.project_slug }}
DB_SSL_MODE ?= disable

# Integration tests always use a separate database (stage testing), even when
# STAGE points to dev. Override it with `make test-integration DB_USER=ci` or
# with TEST_DATABASE_URL. It is the only place in the Makefile holding a DSN.
TEST_DB_NAME ?= {{ cookiecutter.project_slug }}_test
TEST_DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(TEST_DB_NAME)?sslmode=$(DB_SSL_MODE)

# Atlas dev database for `make dbdiff`. By default Atlas starts its own
# Postgres in Docker; override it with ATLAS_DEV_URL to point at another
# database. It never points at the application database.
ATLAS_DEV_URL ?= docker://postgres/16/dev?search_path=public

{% endif %}
fmt:
	go fmt ./...

vet: fmt
	go vet ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint is not installed (https://golangci-lint.run)" >&2; \
		exit 1; \
	fi

build: vet
	go build -o build/http ./cmd/http
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}
	go build -o build/worker ./cmd/worker
{% endif %}

build-all:
	mkdir -p build
	go build -trimpath -ldflags="-s -w" -o build/ ./cmd/...

run:
	STAGE=$(STAGE) go run ./cmd/http

test: test-unit

test-unit:
	STAGE=testing go test ./...

test-integration:
{% if cookiecutter.use_db == "yes" %}
	STAGE=testing TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -tags=integration ./...
{% else %}
	STAGE=testing go test -tags=integration ./...
{% endif %}

coverage:
	STAGE=testing go test -coverprofile=coverage.out ./...

{% if cookiecutter.use_swagger == "yes" %}
docs:
	swag init -g cmd/http/main.go

{% endif %}
{% if cookiecutter.use_db == "yes" %}
# migrate applies the migrations using env/<stage>.env: the URL is built by
# cmd/migrate with net/url, so credentials are not duplicated here.
migrate:
	STAGE=$(STAGE) go run ./cmd/migrate

dbapply: migrate

dbreset:
	STAGE=$(STAGE) go run ./cmd/devcli dropdb
	$(MAKE) migrate STAGE=$(STAGE)

dbdiff:
	STAGE=$(STAGE) atlas migrate diff --env gorm --var dev_url="$(ATLAS_DEV_URL)"

{% endif %}
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}
worker:
	STAGE=$(STAGE) go run ./cmd/worker
{% endif %}

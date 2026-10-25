.PHONY: fmt vet lint build build-all run test test-unit test-integration coverage{% if cookiecutter.use_db == "yes" %} migrate dbreset dbdiff dbapply{% endif %}{% if cookiecutter.use_swagger == "yes" %} docs{% endif %}{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %} worker{% endif %}

# Stage usado por los targets que arrancan la aplicación o la base de datos.
# Hereda STAGE del entorno si ya está definida; una línea de mando
# (make STAGE=testing run) siempre gana sobre el fichero incluido.
STAGE ?= dev

# Carga env/<stage>.env como variables de make para que host, puerto y
# credenciales vivan en un solo sitio (4.2). `-include` no falla si el stage
# no existe: los `?=` de abajo son los fallbacks.
-include env/$(STAGE).env

{% if cookiecutter.use_db == "yes" %}
# Fallbacks por si env/<stage>.env no esté presente.
DB_HOST ?= 127.0.0.1
DB_PORT ?= 5432
DB_USER ?= dev
DB_PASS ?= dev
DB_NAME ?= {{ cookiecutter.project_slug }}
DB_SSL_MODE ?= disable

# Los tests de integración usan siempre una base separada (stage testing),
# aunque STAGE apunte a dev. Se sobreescribe con `make test-integration
# DB_USER=ci` o con TEST_DATABASE_URL. Es el único sitio del Makefile con DSN.
TEST_DB_NAME ?= {{ cookiecutter.project_slug }}_test
TEST_DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(TEST_DB_NAME)?sslmode=$(DB_SSL_MODE)

# Dev-database de Atlas para `make dbdiff`. Por defecto Atlas levanta su
# propio Postgres en Docker; se sobreescribe con ATLAS_DEV_URL para apuntar a
# otra base. Nunca apunta a la base de la aplicación.
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
		echo "golangci-lint no está instalado (https://golangci-lint.run)" >&2; \
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
# migrate aplica las migraciones usando env/<stage>.env: la URL la construye
# cmd/migrate con net/url, así que las credenciales no se duplican aquí.
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

.PHONY: fmt vet build run test test-unit test-integration coverage{% if cookiecutter.use_db == "yes" %} dbreset dbdiff dbapply{% endif %}{% if cookiecutter.use_swagger == "yes" %} docs{% endif %}{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %} worker{% endif %}

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

build: vet
	go build -o build/http ./cmd/http
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}
	go build -o build/worker ./cmd/worker
{% endif %}

run:
	go run ./cmd/http

test: test-unit

test-unit:
	STAGE=testing go test ./...

test-integration:
{% if cookiecutter.use_db == "yes" %}
	STAGE=testing TEST_DATABASE_URL="$${TEST_DATABASE_URL:-postgres://dev:dev@127.0.0.1:5432/{{ cookiecutter.project_slug }}?sslmode=disable}" go test -tags=integration ./...
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
dbreset:
	go run ./cmd/devcli dropdb && \
	atlas migrate apply --url "postgres://dev:dev@127.0.0.1:5432/{{ cookiecutter.project_slug }}?sslmode=disable" --dir file://migrations

dbdiff:
	atlas migrate diff --env gorm

dbapply:
	atlas migrate apply --url "postgres://dev:dev@127.0.0.1:5432/{{ cookiecutter.project_slug }}?sslmode=disable" --dir file://migrations

{% endif %}
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}
worker:
	go run ./cmd/worker
{% endif %}

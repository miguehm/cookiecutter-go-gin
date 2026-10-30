# {{ cookiecutter.project_name }}

{{ cookiecutter.description }}

Generated with `cookiecutter-go-gin`.

## Requirements

- Go 1.27 or newer.
{% if cookiecutter.use_db == "yes" and (cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes") %}
- Docker and Docker Compose (for Postgres and Redis).
{% elif cookiecutter.use_db == "yes" %}
- Docker and Docker Compose (for Postgres).
{% elif cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}
- Docker and Docker Compose (for Redis).
{% endif %}
{% if cookiecutter.use_db == "yes" %}
- Atlas, only if you want to create or apply migrations by hand.
{% endif %}

## Quick start

```sh
{% if cookiecutter.use_db == "yes" or cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}docker compose up -d
{% endif %}go mod tidy
make run
```

The API listens on `http://localhost:8080` and exposes:

- `GET /healthz`: liveness; answers 200 while the process is alive.
- `GET /readyz`: readiness; checks DB and Redis when they are enabled and
  answers 503 naming the dependency that is down.
{% if cookiecutter.use_db == "yes" %}
- `POST /signup`, `POST /login` and `GET /me`: example flow authenticated with
  JWT (`Authorization: Bearer <token>` on `/me`).
{% endif %}

## Configuration

Configuration is loaded from `env/<stage>.env`. The default stage is `dev` and
it can be changed with `STAGE`:

```sh
STAGE=testing go run ./cmd/http
```

Included files:

- `env/dev.env`: local development.
- `env/testing.env`: automated tests.
- `env/prod.env`: production placeholders (fill in the secrets).
- `env/_template.env`: template used by `inv format-env`.
- `.env.example`: copy of every supported variable with safe placeholders.

`configs.Load` reads the file with an isolated Viper instance, so the process
works from any working directory. Environment variables override the file, and
`ENV_FILE`/`ENV_DIR` can point to a different file or directory.

### Environment variables

Variables for features that are disabled in this project are simply ignored.

| Variable | Default | Description |
|---|---|---|
| `STAGE` | `dev` | Environment: `dev`, `testing`, `staging` or `prod`. |
| `PORT` | `8080` | HTTP listen port. |
| `GIN_MODE` | `debug` | Gin mode: `debug`, `release` or `test`. |
| `ENABLE_PPROF` | `false` | Mounts `/pprof` when `true`. |
| `JWT_SECRET` | — | HS256 signing key; at least 32 characters. Required. |
| `JWT_ISSUER` | `go-gin-template` | `iss` claim of issued tokens. |
| `JWT_AUDIENCE` | `go-gin-template` | `aud` claim of issued tokens. |
| `JWT_EXPIRATION` | `24h` | Token lifetime (Go duration). |
| `JWT_LEEWAY` | `30s` | Clock skew tolerance when validating tokens. |
| `BASIC_AUTH_USER` | — | Asynqmon username (required when Asynqmon is enabled). |
| `BASIC_AUTH_PASS` | — | Asynqmon password (required when Asynqmon is enabled). |
| `ENABLE_LOG_REQUEST_DETAIL` | `false` | Logs request bodies (redacted) when `true`. |
| `ENABLE_SWAGGER` | `false` | Mounts `/docs` when `true`. |
| `ENABLE_ASYNQMON` | `false` | Mounts `/asynqmon` when `true`. |
| `ENABLE_HSTS` | `false` | Sends the HSTS header when `true`. |
| `SENTRY_DSN` | — | Sentry DSN; empty disables Sentry. |
| `CORS_ALLOWED_ORIGINS` | `*` outside `prod` | Comma-separated origin allowlist. Required in `prod`. |
| `CORS_ALLOW_CREDENTIALS` | `false` | Allows credentials; cannot be combined with `*`. |
| `TRUSTED_PROXIES` | — | Comma-separated trusted proxy IPs/CIDRs. |
| `MAX_BODY_BYTES` | `1048576` | Maximum request body size. |
| `RATE_LIMIT_ENABLED` | `true` | Enables the in-memory rate limiter. |
| `RATE_LIMIT_RPS` | `5` | Requests per second allowed per client. |
| `RATE_LIMIT_BURST` | `10` | Burst size of the rate limiter. |
| `WORKER_CONCURRENCY` | `10` | Parallel tasks in the Asynq worker. |
| `DB_HOST` | — | Postgres host. |
| `DB_PORT` | `5432` | Postgres port. |
| `DB_USER` | — | Postgres user. |
| `DB_PASS` | — | Postgres password. |
| `DB_NAME` | `{{ cookiecutter.project_slug }}` | Postgres database. |
| `DB_SSL_MODE` | `disable` | Postgres `sslmode`. |
| `DB_LOG_LEVEL` | `4` | GORM log level (1 silent … 4 info). |
| `REDIS_HOST` | — | Redis host. |
| `REDIS_PORT` | `6379` | Redis port. |
| `REDIS_PASS` | — | Redis password. |
| `REDIS_DB` | `0` | Redis database for cache. |
| `REDIS_TASK_DB` | `1` | Redis database for the task queue. |
| `REDIS_ENABLE_SSL` | `false` | Enables TLS for Redis. |

`configs.NewConfig` validates the required values at boot (JWT secret length,
DB/Redis credentials, Asynqmon basic auth, CORS in `prod`, rate limits and
worker concurrency) and `fx` aborts startup if anything is missing.

## API

### Health endpoints

```sh
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/readyz
```

`/healthz` always answers `200 {"message":"success"}`. `/readyz` answers `200`
when every enabled dependency is reachable and `503` with the failing
dependency name otherwise.

{% if cookiecutter.use_db == "yes" %}
### User flow

Create a user:

```sh
curl -X POST http://localhost:8080/signup \
  -H 'Content-Type: application/json' \
  -d '{"firstname":"Ada","lastname":"Lovelace","email":"ada@example.com","password":"supersecret"}'
```

```json
{
  "code": "Success",
  "message": "Success",
  "data": {
    "id": "6f1c0f0e-7a4c-4f7e-9f2a-2b6f0d9c1a11",
    "firstname": "Ada",
    "lastname": "Lovelace",
    "email": "ada@example.com"
  }
}
```

Log in and get a JWT:

```sh
curl -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"supersecret"}'
```

```json
{
  "code": "Success",
  "message": "Success",
  "data": {
    "token": "<jwt>",
    "expires_in": 86400
  }
}
```

Read the current user:

```sh
curl http://localhost:8080/me -H "Authorization: Bearer <jwt>"
```

Validation errors answer `400` with the offending fields:

```json
{
  "code": "ErrBadRequest",
  "message": "Invalid request",
  "details": {
    "email": "Invalid email format",
    "password": "Password must be longer than 8"
  }
}
```

Domain errors use a stable `code` and a user-facing `message`, for example
`401 {"code":"ErrInvalidCredentials","message":"Invalid email or password"}` or
`409 {"code":"ErrEmailTaken","message":"Email already registered"}`.
{% endif %}

{% if cookiecutter.use_swagger == "yes" or cookiecutter.use_asynq == "yes" or cookiecutter.use_pprof == "yes" or cookiecutter.use_sentry == "yes" %}
## Optional features

{% endif %}{% if cookiecutter.use_swagger == "yes" %}
### Swagger

Set `ENABLE_SWAGGER=true` and open `http://localhost:8080/docs/index.html`.
Regenerate the spec with `make docs` (requires `swag`).
{% endif %}
{% if cookiecutter.use_asynq == "yes" %}
### Asynq and Asynqmon

Tasks are enqueued through `internal/core/asynctasks`. The dashboard is mounted
at `http://localhost:8080/asynqmon` when `ENABLE_ASYNQMON=true` and protected
with HTTP basic auth using `BASIC_AUTH_USER`/`BASIC_AUTH_PASS`. The process
refuses to expose it with empty or `dev/dev` credentials.
{% endif %}
{% if cookiecutter.use_pprof == "yes" %}
### pprof

Set `ENABLE_PPROF=true` and use `http://localhost:8080/pprof/`.
{% endif %}
{% if cookiecutter.use_sentry == "yes" %}
### Sentry

Set `SENTRY_DSN` to report errors and warning logs to Sentry. Leave it empty to
disable the integration.
{% endif %}

## Docker

The `Dockerfile` is a multi-stage build that produces static binaries on an
Alpine runtime, runs as a non-root user and starts through `tini`:

```sh
docker build -t {{ cookiecutter.project_slug }}:local .
docker run --rm -p 8080:8080 \
  -e STAGE=prod \
  -e JWT_SECRET="$(openssl rand -hex 32)" \
{% if cookiecutter.use_db == "yes" %}  -e DB_HOST=postgres -e DB_USER=app -e DB_PASS=secret -e DB_NAME={{ cookiecutter.project_slug }} \
{% endif %}{% if cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}  -e REDIS_HOST=redis -e REDIS_PASS=secret \
{% endif %}  {{ cookiecutter.project_slug }}:local
```

The image ships `env/`. `entrypoint.sh` only runs `exec "$@"`, so the server
never applies migrations on startup.
{% if cookiecutter.use_db == "yes" %}
It also ships `migrations/` and the `migrate` binary; apply them as a separate
job or with the Atlas profile:

```sh
docker compose run --rm atlas
```
{% endif %}
`HEALTHCHECK` probes `/healthz` every 30 seconds.

## Project structure

```text
cmd/http/                HTTP API
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}cmd/worker/              Asynq worker
{% endif %}{% if cookiecutter.use_db == "yes" %}cmd/migrate/             applies the Atlas migrations
cmd/devcli/              development utilities (dropdb, fakedb, debugtask)
tools/cmd/atlas_loader/  GORM schema loader for Atlas (tools module)
{% endif %}configs/                 configuration and clients
internal/                controllers, services, repositories and middleware
pkg/                     filters, pagination, converters and cryptography
{% if cookiecutter.use_db == "yes" %}migrations/              Atlas migrations
{% endif %}tools/                   generators and development utilities
```

Tests live next to the code they cover; the empty `tests/` directory is kept so
integration helpers have a home.

## Architecture

- Dependency injection with `fx`: `cmd/http` builds configuration, logger,
  clients and the server, and registers `Start`/`Shutdown` with `fx.Hook`.
- No global state: there are no singletons or `configs.GetDB`; every component
  receives what it needs as a parameter.
- Portable configuration: `configs.Load` reads `env/<stage>.env` with an
  isolated Viper instance and works from any directory and in tests.
- Graceful shutdown: `SIGTERM` closes the HTTP server through `Shutdown` and,
  when a worker exists, drains in-flight tasks.
{% if cookiecutter.use_asynq == "yes" %}
- `cmd/worker` starts Asynq with `WORKER_CONCURRENCY` parallel tasks.
{% endif %}

## Commands

```sh
make fmt                  # go fmt
make vet                  # go vet
make lint                 # golangci-lint run ./...
make build                # builds the binaries
make build-all            # reproducible build of cmd/... (-trimpath -ldflags="-s -w")
make run                  # starts the API
make test                 # alias of test-unit
make test-unit            # tests without external services
make test-integration     # tests with the integration build tag
make coverage             # unit test coverage
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}make worker               # starts the worker
{% endif %}{% if cookiecutter.use_db == "yes" %}make migrate              # applies migrations (uses env/<stage>.env)
make dbreset              # recreates the database and applies migrations
make dbdiff               # generates a migration with Atlas
make dbapply              # alias of migrate
{% endif %}{% if cookiecutter.use_swagger == "yes" %}make docs                 # regenerates Swagger
{% endif %}```

`STAGE` decides which `env/<stage>.env` is used by `make run`{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}, `make worker`{% endif %}{% if cookiecutter.use_db == "yes" %}, `make migrate` and `make dbreset`{% endif %}.
{% if cookiecutter.use_db == "yes" %}
The Makefile injects the same credentials into `TEST_DATABASE_URL` for
`make test-integration`. The `make dbdiff` dev database is independent and is
controlled by `ATLAS_DEV_URL` (by default the Postgres that Atlas starts in
Docker).
{% endif %}

## Tests

The recommended strategy is:

- Unit tests without external services: filters, pagination, JWT and converters.
- Handlers with `httptest` and `gin.TestMode`.
- Services with repository fakes.
- Repository integration tests behind the `integration` build tag.

```sh
go test ./...
go test -race ./...
go test -tags=integration ./...
```

Integration tests are skipped when Docker is not available.

## Migrations

{% if cookiecutter.use_db == "yes" %}
```sh
make dbdiff                        # atlas migrate diff --env gorm
make migrate                       # atlas migrate apply with env/<stage>.env
make dbdiff ATLAS_DEV_URL="postgres://dev:dev@127.0.0.1:5432/atlas?sslmode=disable"
```

`cmd/migrate` builds the URL with `net/url` from `env/<stage>.env`, so you do
not need to write credentials in the Makefile or on the command line.
{% else %}
This project was generated without a database; it does not include
`migrations/`, `cmd/migrate` or `tools/cmd/atlas_loader`.
{% endif %}

## Development utilities

```sh
{% if cookiecutter.use_db == "yes" %}go run ./cmd/devcli dropdb          # drops every table and the Atlas schema
go run ./cmd/devcli fakedb          # seeds fake data (extend it as needed)
{% endif %}{% if cookiecutter.use_asynq == "yes" %}go run ./cmd/devcli debugtask       # enqueues a debug task
{% endif %}{% if cookiecutter.use_db == "yes" %}uvx --from invoke --with jinja2 inv gen-model <ModelName>
{% endif %}uvx --from invoke --with jinja2 inv format-env --stage <stage|all>
```

{% if cookiecutter.use_db == "yes" %}`gen-model` creates the model, repository and service. {% endif %}`format-env` renders `env/_template.env` onto the file of the given stage.

{% if cookiecutter.use_db == "yes" %}
### Adding a model and an endpoint

1. Generate the model, repository and service:
   `uvx --from invoke --with jinja2 inv gen-model Invoice`.
2. Add the fields and tags to `internal/core/models/invoice.go`.
3. Create the migration with `make dbdiff` and apply it with `make migrate`.
4. Register the repository, service and controller in `cmd/http/main.go`
   (`fx.Provide`) and mount the routes in `internal/server/router.go`.
5. Add request/response value objects in `internal/core/vo` and a controller in
   `internal/controllers`.
{% endif %}

## Continuous integration

The generated `.github/workflows/ci.yml` runs on push, pull requests and
manually with the following jobs:

- `test`: `go test -race ./...` and a coverage artifact.
- `security`: `govulncheck`, `gosec`, `gitleaks` and `trivy`, publishing SARIF.
{% if cookiecutter.use_db == "yes" or cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}
- `integration`: Postgres/Redis services and `go test -tags=integration ./...`.
{% endif %}

Reproduce it locally with:

```sh
make test-unit
make test-integration
make lint
```

## Deployment

1. Fill in `env/prod.env` (never commit real secrets) or provide the same
   variables through the environment.
2. Set `STAGE=prod`. The configuration requires a non-empty
   `CORS_ALLOWED_ORIGINS` and a strong `JWT_SECRET`, and disables Swagger,
   Asynqmon and pprof by default.
3. Build and push the image.
{% if cookiecutter.use_db == "yes" %}4. Apply migrations as a separate job with the `migrate` binary or the Atlas
   image before rolling out the new version.
{% endif %}5. Probe `/healthz` for liveness and `/readyz` for readiness.

## License

{% if cookiecutter.license == "proprietary" %}
This project is proprietary and confidential. See the `LICENSE` file.
{% else %}
This project is distributed under the `{{ cookiecutter.license }}` license. The
full text is in `LICENSE`.
{% endif %}

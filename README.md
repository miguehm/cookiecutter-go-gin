# cookiecutter-go-gin

[Cookiecutter](https://github.com/cookiecutter/cookiecutter) template for
creating Go projects with Gin, `fx` dependency injection, the repository +
service pattern and the utilities a small or medium backend usually needs.

## Requirements

- [uv](https://docs.astral.sh/uv/) (includes `uvx`).
- Go 1.27 or newer.
- Docker and Docker Compose for the test harness and for the services of the
  projects that enable a database, Redis or Asynq.
- Optional: Atlas, `golangci-lint`, `govulncheck`, `gosec`, `gitleaks` and
  `trivy`.

Do not install Cookiecutter with `pip`: the version is pinned in the `Makefile`
and in CI, and it is always executed with `uvx`. To read it:

```sh
make cookiecutter-version
```

## Usage

Interactive generation (asks every variable):

```sh
version="$(make -s cookiecutter-version)"
uvx --from "cookiecutter==$version" cookiecutter .
```

Generation without questions using one of the harness fixtures:

```sh
version="$(make -s cookiecutter-version)"
uvx --from "cookiecutter==$version" cookiecutter --no-input \
  --config-file tests/fixtures/full.yaml \
  --output-dir /tmp \
  .
```

Without `--config-file`, `--no-input` uses the defaults from
`cookiecutter.json`: `project_name` «Project Name» (slug `projectname`), module
`github.com/yourname/projectname`, MIT license and every feature enabled.

## Variables

| Variable | Description |
|---|---|
| `project_name` | Human-readable project name. |
| `project_slug` | Directory name and database name. |
| `description` | Short description. |
| `author`, `email` | Author and email; used in `LICENSE` and metadata. |
| `go_module` | Explicit Go module, for example `github.com/acme/myapp`. |
| `license` | `MIT`, `Apache-2.0`, `BSD-3-Clause` or `proprietary`. |
| `use_db` | Includes GORM, Postgres, Atlas, `cmd/migrate`, `cmd/devcli`, `tools/` and `migrations/`. |
| `use_redis` | Includes the Redis client and the service in `compose.yaml`. |
| `use_asynq` | Includes Asynq, Asynqmon, the tasks and the Redis client. |
| `use_worker` | Includes `cmd/worker`. Requires `use_asynq=yes`. |
| `use_swagger` | Includes Swagger at `/docs`. |
| `use_pprof` | Includes pprof at `/pprof`. |
| `use_sentry` | Includes Sentry in logs and middleware. |

The flags use `yes`/`no` and are validated in `hooks/pre_gen_project.py`.
`hooks/post_gen_project.py` removes the files that do not apply and formats the
result with `gofmt`.

## Optional features

| Flag | Default | What it adds | Requirements |
|---|---|---|---|
| `use_db` | `yes` | GORM + Postgres, Atlas, migrations and development utilities. | Docker for the smoke test. |
| `use_redis` | `yes` | Cache client and Redis service. | Docker for the smoke test. |
| `use_asynq` | `yes` | Task queue, dashboard at `/asynqmon` and Redis client. | Docker for the smoke test. |
| `use_worker` | `yes` | `cmd/worker` with configurable concurrency. | `use_asynq=yes`. |
| `use_swagger` | `yes` | Documentation at `/docs/index.html` and `make docs`. | `swag` to regenerate. |
| `use_pprof` | `yes` | Profiles at `/pprof` when `ENABLE_PPROF=true`. | — |
| `use_sentry` | `yes` | Error capture and warning logs. | Optional `SENTRY_DSN`. |

## After generating

The generated project ships its own `README.md` with the full usage guide
(environment variables, API examples, Docker, features, CI and deployment).
The short version:

```sh
cd <project_slug>
docker compose up -d   # only if the project enables Postgres/Redis/Asynq
go mod tidy
make run
```

Then:

- Read `env/<stage>.env` and replace `JWT_SECRET` and the database/Redis
  credentials. `.env.example` lists every supported variable with safe
  placeholders.
- Open `http://localhost:8080/healthz` and `http://localhost:8080/readyz`.
- If the project uses a database, apply the migrations with `make migrate` (or
  `docker compose run --rm atlas`) before using the example user endpoints.
- Run the checks with `make test-unit`, `make test-integration` and `make lint`.

The generated project does not have a `cookiecutter-version` target; that target
belongs to this template repository.

## Generated structure

```text
cmd/http/                HTTP API
cmd/worker/              Asynq worker (if use_worker=yes)
cmd/devcli/              development utilities (if use_db=yes)
cmd/migrate/             applies Atlas migrations (if use_db=yes)
tools/cmd/atlas_loader/  GORM schema loader for Atlas (if use_db=yes)
tools/templates/         Jinja generators for model, repository and service
configs/                 configuration and clients
internal/                controllers, services, repositories and middleware
pkg/                     filters, pagination, converters and cryptography
migrations/              Atlas migrations (if use_db=yes)
tests/                   generated project tests
```

## License

The generated project includes a `LICENSE` with the full text of the selected
license:

- `MIT`, `Apache-2.0` and `BSD-3-Clause`: standard text with the author and the
  generation year.
- `proprietary`: copyright notice and confidentiality clause, without any open
  license text.

The template itself is distributed under the repository license (see `LICENSE`
at the root).

## Renovate

`renovate.json` groups updates for:

- Go modules (`go.mod` and `tools/go.mod`);
- `Dockerfile` and `compose.yaml` images;
- GitHub Actions;
- CI tool versions (`golangci-lint`, `govulncheck`, `gosec`, `gitleaks`,
  `trivy`);
- the Cookiecutter pin in the `Makefile`.

Versions live in a single place per tool; fixtures and documentation read them
with `make cookiecutter-version` or with the `Makefile` variables.

## Template test harness

The harness lives in `tests/` (template root, not inside the generated project)
and uses `uvx` to invoke the real Cookiecutter CLI.

```sh
make test-template
FIXTURES=minimal,full make test-template   # reduced matrix
```

Each fixture is generated with the pinned version:

```sh
version="$(make -s cookiecutter-version)"
uvx --from "cookiecutter==$version" cookiecutter --no-input \
  --config-file tests/fixtures/<fixture>.yaml \
  --output-dir "$tmp" .
```

On the generated project it runs `go mod tidy -diff`, `go build ./...`,
`go vet ./...`, `go test ./...`, `go test -race ./...`, `golangci-lint run`,
`govulncheck ./...`, `gosec -quiet ./...` and `gitleaks detect`. The harness may
use the network and cache `GOMODCACHE` and `UV_CACHE_DIR`; generation hooks do
not use the network.

`golangci-lint`, `govulncheck`, `gosec` and `gitleaks` must be in `PATH`; if
they are missing, their tests are skipped. CI pins the versions and enables
`REQUIRE_GOVULNCHECK=1` and `REQUIRE_SECURITY=1` so a missing tool is a failure,
not a skip:

```sh
REQUIRE_GOVULNCHECK=1 REQUIRE_SECURITY=1 make test-template
make security   # gitleaks + trivy over the template code
```

## Security

What the template applies by default in the generated project:

- `configs.NewConfig()` loads and validates the configuration at boot (`STAGE`,
  `JWT_SECRET` with a minimum length, DB/Redis credentials, `BASIC_AUTH_*` when
  Asynqmon is enabled, CORS in `prod` and `WORKER_CONCURRENCY`) and returns an
  error if anything is missing; `fx` aborts startup. No library uses
  `log.Fatal`: they return errors and `main` decides.
- Per-environment CORS (`CORS_ALLOWED_ORIGINS`/`CORS_ALLOW_CREDENTIALS`), failing
  closed in `prod` and never combining `*` with credentials.
- Security headers (`X-Frame-Options`, CSP, optional HSTS), body limit with
  `http.MaxBytesReader`, explicit `TrustedProxies` and full server timeouts.
- In-memory rate limit (`golang.org/x/time/rate`) with `/healthz` exempt.
- HS256 JWT with configurable `alg`, `iss`, `aud`, `iat`, `jti`, expiration and
  leeway through `JWT_*`.
- Structured logs with `slog`, `X-Request-ID` and redaction of
  `password`/`token`/`authorization`/`cookie` in bodies and query strings.
- Migrations outside startup: `CMD ["./http"]` and a `./migrate` binary (plus
  the `integration` CI job) that applies `atlas migrate apply` with the URL
  built through `net/url`.
- The generated CI includes `gosec`, `govulncheck`, `gitleaks` and `trivy`
  pinned by version and publishes SARIF.

## Useful commands

```sh
make help                 # help with the pinned versions
make cookiecutter-version # prints the Cookiecutter pin
make test-template        # generates every fixture and validates it
make security             # gitleaks + trivy over the template code
```

# Upgrade plan — cookiecutter-go-gin

Goal: make the template generate a project that builds, is up to date, secure and
has a clear testing strategy, and improve its DX. Run the phases in order; each
phase must end with the test harness green (`uvx cookiecutter` + `go build ./...`
+ `go vet ./...` + `go test ./...`).

## Conventions

- All trial generations use `uvx`; never `pip install cookiecutter`.
- The cookiecutter version is pinned in a single place (`COOKIECUTTER_VERSION` in
  the `Makefile` and in CI) and updated by Renovate. Do not mix a range
  (`>=2.6,<3`) with an exact pin.
- Canonical command:

  ```sh
  uvx --from "cookiecutter==${COOKIECUTTER_VERSION}" cookiecutter --no-input \
    --config-file tests/fixtures/<fixture>.yaml \
    --output-dir "$tmp" .
  ```

- Tests invoke `uvx cookiecutter` via `subprocess`, using the same pinned
  version (`--from`); do not use `pytest-cookies`, because it runs cookiecutter
  in-process and does not exercise the real CLI.
- Test dependencies are declared explicitly:

  ```sh
  uvx --with pytest --with pyyaml --with pexpect pytest -q tests/
  ```

- Hooks may only use the Python stdlib. If they need packages, declare them
  explicitly with `uvx --with`.
- The test harness may use the network (Go module downloads); cache
  `GOMODCACHE` and `UV_CACHE_DIR`. Generation hooks must not depend on the
  network.
- Required tools: Go (version from the matrix), uv/uvx, Docker and Atlas.
  For lint and security: `golangci-lint`, `govulncheck`, `gosec` and `trivy`,
  pinned by version in CI and the Makefile.
- The `file:line` references are indicative; validate them against the code
  before editing.
- Step IDs (`Phase.Step`; in Phase 0 the subphase is added, e.g. `0.1.4`) are
  stable and are used in the dependency tables.

## Tracking

### States

- `[ ]` pending · `[~]` in progress · `[x]` done and verified · `[!]` blocked ·
  `[–]` discarded.

A step only becomes `[x]` once its evidence is recorded in its subphase's table
and the phase gate is still green. `[~]`, `[!]` and `[–]` do not render as
checkboxes on GitHub, but are readable as raw text.

### Progress

| Phase | Steps | Done | Blocked | Gate |
|---|---:|---:|---:|---|
| 0 | 49 | 49 | 0 | `make test-template` |
| 1 | 7 | 7 | 0 | `go mod tidy -diff` + reproducible build |
| 2 | 20 | 20 | 0 | `govulncheck`/`gosec`/`gitleaks` green |
| 3 | 8 | 8 | 0 | no globals or `log.Fatal` in libraries |
| 4 | 8 | 8 | 0 | full CI over the matrix |
| 5 | 8 | 8 | 0 | interactive mode + `--no-input` |
| **Total** | **100** | **100** | **0** | — |

### Phase dependencies

| Phase | Depends on | Unlocks |
|---|---|---|
| 0 | — | 1, 2, 3, 4, 5 |
| 1 | 0.1, 0.3 | 2, 3, 4 |
| 2 | 1, 0.3, 0.4 | 3, 4 |
| 3 | 0.1, 1, 2 | 4 |
| 4 | 0.3, 0.4, 1, 3 | 5 |
| 5 | 0.2, 0.3 | — |

## Phase 0 — Reproducible, buildable base and uvx harness

### 0.1 Build and startup

#### Dependencies

| Step | Depends on | Unlocks | Evidence |
|---|---|---|---|
| 0.1.1 | — | 0.1.2 | baseline commit + initial `go build ./...` |
| 0.1.2 | 0.1.1 | 0.1.3 | `go build ./...` |
| 0.1.3 | 0.1.2 | 0.1.4, 0.1.5 | `go build ./...` without errors |
| 0.1.4 | 0.1.3 | 0.1.5 | `Bootstrap` referenced or removed |
| 0.1.5 | 0.1.3, 0.1.4 | 0.1.6, 0.1.17 | deterministic startup |
| 0.1.6 | 0.1.5 | 0.1.17 | logger/startup test |
| 0.1.7 | 0.1.2 | 0.1.17 | `go build ./...` |
| 0.1.8 | 0.1.2 | 0.1.17 | JWT test |
| 0.1.9 | 0.1.3 | 0.1.17 | `go vet ./...` |
| 0.1.10 | 0.1.2 | 0.1.17 | `go build ./...` |
| 0.1.11 | 0.1.2 | 0.1.17 | `make` without errors |
| 0.1.12 | 0.1.2 | 0.1.17 | `make help` |
| 0.1.13 | 0.1.2 | 0.1.17 | render of `model.j2` |
| 0.1.14 | 0.1.2 | 0.1.17 | `go vet ./...` |
| 0.1.15 | 0.1.2 | 0.1.17 | `go test ./pkg/filters/...` |
| 0.1.16 | 0.1.2 | 0.1.17 | build or `docker build` |
| 0.1.17 | 0.1.2–0.1.16 | 1.1 | `go build ./...` + `go vet ./...` |

#### Checklist

- [x] **0.1.1** Create a reference/baseline commit and run `go build ./...`
      on a generated project to record the initial state.
- [x] **0.1.2** Add the imported, missing dependencies to `go.mod`:
      `go.uber.org/fx`, `github.com/gin-contrib/pprof` and
      `github.com/hibiken/asynqmon`.
- [x] **0.1.3** Fix compile errors not covered by the initial review:
  - `cmd/http/main.go:20`: `configs.PreServerShutdown()` does not exist; it
    must use `configs.PreServerStop()` or the fx hook.
  - `cmd/http/main.go:25`: `configs.AsynqClient` is private; close it with
    `configs.GetAsynqClient().Close()` or register the close in an `fx.Hook`.
  - Guarantee `InitLogger()` before `GetLogger()`: today no `main` calls it
    and `internal/middlewares/error_handler.go:18` invokes it during package
    initialization.
- [x] **0.1.4** Add `configs.Bootstrap()` or remove the calls at
      `cmd/worker/main.go:16` and `cmd/devcli/main.go:15`.
- [x] **0.1.5** Initialize `InitLogger`, `InitDBClient`, `InitRedisClient` and
      `InitAsynqClient` deterministically, in the correct order.
- [x] **0.1.6** Replace `var logger = configs.GetLogger()` with lazy
      initialization in `internal/middlewares/error_handler.go:18`.
- [x] **0.1.7** Fix the Gin route `docs/*any` → `/docs/*any` at
      `internal/server/router.go:49`.
- [x] **0.1.8** Avoid the `fields[1]` panic at
      `internal/middlewares/jwt_auth.go:23` (validate the `Bearer` scheme and
      length).
- [x] **0.1.9** Remove the double call to `PreServerStart()` in
      `cmd/http/main.go`.
- [x] **0.1.10** Fix `configs.GetAsynqClient.Enqueue` →
      `configs.GetAsynqClient().Enqueue(...)` at
      `internal/core/asynctasks/common.go:40`.
- [x] **0.1.11** Fix the line with spaces at `Makefile:25` (use a tab).
- [x] **0.1.12** Remove/implement the `mon` target (`cmd/asynqmon` does not
      exist) and drop the `push` target hardcoded to `unstable`.
- [x] **0.1.13** Fix `tools/templates/model.j2` (avoid duplicate `gorm.Model` +
      `ID uuid.UUID`).
- [x] **0.1.14** Fix `pkg/pagination/pagination_test.go`: it imports
      `internal/models` and `models.Agency`, which do not exist, and makes
      `go vet ./...` fail.
- [x] **0.1.15** Add real assertions to `pkg/filters/filters_test.go` (it
      currently verifies nothing).
- [x] **0.1.16** Create `migrations/` (with `.gitkeep` or an initial migration)
      and the `go.sum` expected by the `Dockerfile`; or adjust the
      `Dockerfile` if we decide not to track them.
  - **Decision:** track `migrations/` and `go.sum`, or adjust the `Dockerfile`;
    record the chosen option in this step.
- [x] **0.1.17** Verify that `go build ./...` and `go vet ./...` pass without
      external services.

### 0.2 Configuration and template

#### Dependencies

| Step | Depends on | Unlocks | Evidence |
|---|---|---|---|
| 0.2.1 | — | 0.2.2, 0.2.3, 0.2.9, 0.2.12, 5.2, 5.5 | valid `cookiecutter.json` |
| 0.2.2 | 0.2.1 | 0.2.4, 0.2.5, 0.2.7, 0.2.8, 0.3.1, 0.3.5, 5.1 | conditional fixtures |
| 0.2.3 | 0.2.1 | 0.2.4, 0.2.10 | `go_module` in generated project |
| 0.2.4 | 0.2.2, 0.2.3 | 0.2.10, 0.3.6 | `pre_gen` negative tests |
| 0.2.5 | 0.2.2 | 0.2.6 | env vs `EnvConfig` diff |
| 0.2.6 | 0.2.5 | 0.2.12 | no versioned secrets |
| 0.2.7 | 0.2.2 | 0.2.11 | `tasks.py` works |
| 0.2.8 | 0.2.7 | 0.2.11 | stages aligned |
| 0.2.9 | 0.2.1 | 0.3.3 | `go mod tidy -diff` |
| 0.2.10 | 0.2.3, 0.2.4 | 5.4 | minimal hook green |
| 0.2.11 | 0.2.7, 0.2.8 | 0.2.12 | valid README references |
| 0.2.12 | 0.2.1, 0.2.6, 0.2.11 | 5.6 | complete root README |

#### Checklist

- [x] **0.2.1** Consolidate all `cookiecutter.json` changes here (removing the
      duplication with Phase 5): `go_module`, author, email, license,
      `version`, and the DB/Redis/Asynq/Worker/Swagger/pprof/Sentry flags.
- [x] **0.2.2** Define the flags in this phase, before the harness, so the
      fixtures can test the conditional files.
- [x] **0.2.3** Use an explicit Go module (`go_module`) instead of
      `project_slug` as the module path.
- [x] **0.2.4** Add `hooks/pre_gen_project.py` validating `project_slug` and
      `go_module` (no spaces, no empty segments, no `..`, no leading digit, no
      Go reserved words).
- [x] **0.2.5** Align `env/_template.env`, `env/dev.env` and
      `configs.EnvConfig`: `JWT_SECRET`, `DB_NAME`, `REDIS_DB`,
      `REDIS_TASK_DB`, `ENABLE_LOG_REQUEST_DETAIL`, `STAGE`, etc. are missing.
- [x] **0.2.6** Stop versioning credentials in `env/dev.env`; use non-sensitive
      development values and document `.env.example`.
- [x] **0.2.7** Fix `tasks.py`: it reads `env/template.env` but the file is
      `env/_template.env`, and it renders `{{ .PORT }}` with Jinja (invalid
      syntax). Unify the name and the template syntax.
- [x] **0.2.8** Align the `tasks.py` stages (`dev`, `testing`, `unstable`,
      `staging`) with the real files, or remove the ones that do not exist.
- [x] **0.2.9** Decide the `go.sum` strategy: include it in the template and
      verify with `go mod tidy -diff`; document that the harness may use the
      network and cache modules, while hooks must not.
  - **Decision:** include `go.sum` in the template and verify it with
    `go mod tidy -diff`; record the chosen option in this step.
- [x] **0.2.10** Reduce `hooks/post_gen_project.py` to a minimal, Python 3
      hook: format with `gofmt`, validate the result and show next steps.
      Remove all the legacy Cookiecutter Django code.
- [x] **0.2.11** Fix the template README references (`cmd/atlas_loader`,
      `env/_template.env`, `invoke`) and document the `uvx` flow.
- [x] **0.2.12** Update the root README: requirements, `uvx cookiecutter`,
      structure, generated commands, and use of optional features.

### 0.3 Test harness and minimal CI

#### Dependencies

| Step | Depends on | Unlocks | Evidence |
|---|---|---|---|
| 0.3.0 | 0.1.17 | 0.3.7, 3.8, 4.2 | `docker compose config` |
| 0.3.1 | 0.2.2 | 0.3.2, 0.3.3, 5.8 | fixtures created |
| 0.3.2 | 0.3.1 | 0.3.3, 0.3.8, 4.1 | `make test-template` |
| 0.3.3 | 0.2.9, 0.3.1, 0.3.2 | 0.3.4, 0.3.7, 0.3.9 | local matrix green |
| 0.3.4 | 0.3.3, 0.4.6 | 0.3.8 | `go test ./...` without services |
| 0.3.5 | 0.2.2, 0.3.1 | 0.3.8 | conditional-file asserts |
| 0.3.6 | 0.2.4, 0.3.1 | 0.3.8 | hook negative tests |
| 0.3.7 | 0.3.0, 0.3.3 | 0.3.8 | `/healthz` smoke test |
| 0.3.8 | 0.3.2–0.3.7 | 2.19, 4.5, 5.8 | minimal CI green |
| 0.3.9 | 0.3.3 | 0.3.8 | harness README |

#### Checklist

- [x] **0.3.0** Add `compose.yaml` with Postgres, Redis and Atlas (farewell to
      `devhost`). Moved from the old 4.1 so the 0.3.7 smoke test does not
      depend on a later phase.
- [x] **0.3.1** Create `tests/` at the template repository root (not inside
      `{{cookiecutter.project_slug}}/`) with `test_template.py` and fixtures
      `tests/fixtures/minimal.yaml`, `no-db.yaml`, `no-redis.yaml`,
      `no-asynq.yaml`, `no-worker.yaml`, `no-swagger.yaml`, `no-sentry.yaml`
      and `full.yaml`.
- [x] **0.3.2** Add a root `Makefile` with `COOKIECUTTER_VERSION` and
      `test-template: uvx --with pytest --with pyyaml --with pexpect pytest -q
      tests/`.
- [x] **0.3.3** The test must generate each fixture with `uvx cookiecutter`,
      locate the generated directory and run against it: `go mod download`,
      `go mod tidy -diff`, `go build ./...`, `go vet ./...`, `go test ./...`,
      `golangci-lint run` and `govulncheck ./...`.
- [x] **0.3.4** Run `go test ./...` without external services by default;
      integration tests go behind a build tag (see 0.4.6).
- [x] **0.3.5** Check that conditional files exist or not according to the
      flags (`DB`, `Redis`, `Asynq`, `Worker`, `Swagger`, `pprof`, `Sentry`).
- [x] **0.3.6** Add negative tests for `pre_gen_project.py`: invalid slugs and
      modules must fail with a clear message.
- [x] **0.3.7** Add a startup smoke test when the fixture requires DB/Redis:
      `compose up`, wait for healthchecks, check `/healthz` and tear down. It
      must `skip` when Docker is unavailable.
- [x] **0.3.8** Add minimal CI from this phase on (see Phase 4 for the full
      CI): install Go and `astral-sh/setup-uv`, cache `UV_CACHE_DIR` and
      `GOMODCACHE`, install the pinned tools and run
      `uvx --with pytest --with pyyaml --with pexpect pytest -q tests/`.
- [x] **0.3.9** Document the harness in the root README, including that trial
      generations use `uvx`.

### 0.4 Testing strategy for the generated project

#### Dependencies

| Step | Depends on | Unlocks | Evidence |
|---|---|---|---|
| 0.4.1 | — | 0.4.7 | README with the test pyramid |
| 0.4.2 | 0.4.7 | 0.4.3, 0.4.6, 0.4.8, 2.20 | unit `go test ./...` |
| 0.4.3 | 0.4.2 | 0.4.6 | handler tests with `httptest` |
| 0.4.4 | 0.4.2 | 0.4.6 | service tests with fakes |
| 0.4.5 | 0.4.7 | 0.4.6 | integration with `t.Skip` |
| 0.4.6 | 0.4.2–0.4.5 | 0.3.4, 0.4.8, 0.4.9, 0.4.10 | `integration` tag |
| 0.4.7 | 0.4.1 | 0.4.2, 0.4.5 | helpers in `internal/testutil` |
| 0.4.8 | 0.4.6 | 4.1, 4.4 | `test*` targets |
| 0.4.9 | 0.4.6 | 4.4, 4.5 | generated project CI |
| 0.4.10 | 0.4.6 | 2.20, 4.4 | unit/integration split |

#### Checklist

- [x] **0.4.1** Define the test pyramid and the commands to run it in the
      generated project's README.
- [x] **0.4.2** Pure unit tests (no DB/Redis): filters, pagination, JWT,
      config validation and converters. Case tables, `t.Parallel()` and
      `go test -race`.
- [x] **0.4.3** Handlers with `httptest` + `gin.TestMode`, verifying codes,
      body and errors.
- [x] **0.4.4** Services with fakes of the already existing repository
      interfaces.
- [x] **0.4.5** Repositories only in integration, with `testcontainers-go` or
      `compose`: schema/DB isolated per test, `t.Cleanup` and `t.Skip` when
      Docker is unavailable.
- [x] **0.4.6** Isolate integration with the `integration` build tag so
      `go test ./...` passes without external services.
- [x] **0.4.7** Create helpers in `internal/testutil` (fixtures, injected
      config, factories) and avoid depending on `devhost` or global state.
- [x] **0.4.8** Add `test`, `test-unit`, `test-integration` and `coverage`
      targets to the generated Makefile.
- [x] **0.4.9** Include a CI workflow in the generated project with
      `go test -race ./...`, coverage and the integration job.
- [x] **0.4.10** Replace the current `test-resetdb` (which demands DB and Atlas
      for any test) with an explicit unit/integration split.

Phase 0 acceptance criteria: `make test-template` generates all fixtures; each
project builds, passes `go vet`, passes `go test` without external services,
passes `golangci-lint` and `govulncheck`, and starts with Postgres/Redis
(depending on its flags) passing the `/healthz` smoke test.

#### Phase 0 evidence

- `make test-template`: 42 passed, 9 skipped (missing tools and `compose.yaml`
  in `minimal`) in 61 s.
- Harness with `golangci-lint` and `govulncheck` installed: 42 passed, 1
  skipped, 8 `xfail`. `golangci-lint` is clean across all 8 fixtures.
  `govulncheck` flags vulnerabilities from old dependencies and the local
  stdlib; left as a documented `xfail` until Phase 1 (`REQUIRE_GOVULNCHECK=1`
  makes it blocking).
- `full` smoke test: Postgres and Redis from `compose.yaml`, API up at
  `/healthz`, `docker compose down -v` in `finally`.
- Model generation with `inv`: `HarnessModel` produces model, repository and
  service, and the project keeps building.
- The remote GitHub Actions workflow could not be run from this environment;
  the harness and generated YAML are validated with PyYAML in the tests.

## Phase 1 — Upgrade

### Dependencies

| Step | Depends on | Unlocks | Evidence |
|---|---|---|---|
| 1.1 | 0.1.17 | 1.2 | `go build ./...` with Go 1.27 |
| 1.2 | 1.1 | 1.3, 5.7 | `go list -m -u all` |
| 1.3 | 1.2 | 1.4, 1.6, 1.7 | `go mod tidy -diff` |
| 1.4 | 1.3 | 1.5 | final image decision recorded |
| 1.5 | 1.4 | 1.7, 2.17 | `docker build` |
| 1.6 | 1.3 | 1.7 | build of `tools/` |
| 1.7 | 1.3, 1.5, 1.6 | 2.1 | reproducible build + `go mod tidy -diff` |

### Checklist

- [x] **1.1** Bump `go` to 1.27 and the base image to `golang:1.27-alpine`
      (pinned by digest). Verify availability before pinning the version; if
      it does not exist, use the latest stable and update this plan.
  - **Executed:** `go 1.27.3`, `golang:1.27-alpine@sha256:b8c9b82a…`,
    `alpine:3.22.7@sha256:2e0b81d6…`, root CI and generated CI aligned.
- [x] **1.2** Update dependencies, verifying every jump with
      `go list -m -u all` and `go get`; do not pin versions blindly. Initial
      reference versions:
  - `gin 1.10.0 → 1.12.0`
  - `gorm 1.25.12 → 1.31.2`, `driver/postgres 1.5.11 → 1.6.3`
  - `atlas-provider-gorm 0.5.0 → 0.6.1`
  - `asynq 0.25.1 → 0.26.0` + `asynqmon 0.7.2`
  - `viper 1.19.0 → 1.21.0`
  - `golang-jwt/jwt/v5 5.2.1 → 5.3.1`
  - `go-redis/v9 9.7.0 → 9.23.0`
  - `sentry-go 0.30.0 → 0.50.0`
  - `fx → 1.24.0`
  - Also: `cors 1.7.9`, `pprof 1.5.6`, `validator/v10 10.30.5`,
    `gin-swagger 1.6.1`, `slog-multi 1.8.0`, `slog-sentry/v2 2.11.0`,
    `urfave/cli/v2 2.27.7`, `x/crypto 0.58.0`, `decimal 1.6.0` and
    `sentry-go/gin 0.50.0`.
- [x] **1.3** Run `go mod tidy` and remove indirect dependencies no longer in
      use (MySQL/SQLServer/SQLite drivers if Atlas is split out).
  - **Executed:** root `go.sum` and `tools/go.sum` regenerated from the `full`
    fixture; `go mod tidy -diff` clean in both modules.
- [x] **1.4** Resolve the final-image contradiction: `distroless/static` has no
      shell, so it cannot run `entrypoint.sh`. Choose one option: keep a base
      with a shell, use a Go binary for startup, or move migrations to a
      separate service/job and point `CMD` at the binary. Consider
      `tini`/`dumb-init` as PID 1 if an entrypoint is kept.
  - **Decision:** base with shell (`alpine:3.22.7` pinned by digest) + `tini`
    as PID 1 and `entrypoint.sh` running the migration. Distroless will be
    re-evaluated in 2.17, when migrations move to a separate job.
- [x] **1.5** Multi-stage Dockerfile:
  - non-root user;
  - minimal final image, pinned by digest;
  - `-trimpath -ldflags="-s -w"`;
  - `ARG TARGETARCH` instead of a fixed `GOARCH=amd64`;
  - Atlas pinned with checksum;
  - `HEALTHCHECK` and multiarch build with `buildx`.
- [x] **1.6** Split `cmd/atlas_loader` into `tools/go.mod` so it does not drag
      in the MySQL/SQLServer/SQLite drivers, and update `atlas.hcl`
      accordingly (e.g. `go -C tools run ./cmd/atlas_loader`). Add `go.work`
      if both modules coexist.
  - **Decision:** nested module `{{ cookiecutter.go_module }}/tools` with a
    `replace` to the root module and `go -C tools`; no `go.work`. The hook
    removes `tools/cmd`, `tools/go.mod` and `tools/go.sum` when
    `use_db=no`.
- [x] **1.7** Verify `go mod tidy -diff` and that the build is reproducible.

Acceptance criteria: `go list -m -u all` with no relevant updates,
`go mod tidy -diff` clean and reproducible build.

#### Phase 1 evidence

- `make test-template` with `REQUIRE_GOVULNCHECK=1`: **50 passed, 1 skipped**
  (`compose` smoke in `minimal`). `golangci-lint v2.15.0` clean across the 8
  fixtures and `govulncheck` without exploitable vulnerabilities.
- `go mod tidy -diff` clean in the root module and in `tools` for the `full`
  fixture; `go -C tools build ./...` and `go -C tools vet ./...` clean.
- Reproducible build: two builds of `cmd/http` with
  `-trimpath -buildvcs=false -ldflags="-s -w"` produce the same SHA-256
  (`06acd74a…`).
- `go -C tools run ./cmd/atlas_loader` generates the SQL schema of `users`.
- `docker build` of the `full` fixture succeeds on `linux/amd64` (Atlas 1.3.5
  verified by checksum). The `linux/arm64` build could not run locally for
  lack of emulation/binfmt; it is covered by the Phase 4 multiarch CI.
- Fixed `pkg/pagination/pagination.go:63` (`Statement.DB` → `Statement`) to
  keep `golangci-lint` green with `gorm 1.31.2`.
- Pending updates in `go list -m -u all`: only transitive indirects; all direct
  dependencies are at their latest version.

## Phase 2 — Security

### Dependencies

| Step | Depends on | Unlocks | Evidence |
|---|---|---|---|
| 2.1 | 0.1.17 | 2.2, 2.3, 2.20 | table-driven filter tests |
| 2.2 | 2.1 | 2.20 | test of `uuid.Parse` with invalid bytes |
| 2.3 | 2.1 | 2.20 | test of invalid filter with error |
| 2.4 | 0.1.17 | 2.5, 2.20 | test of `page`/`size` limits |
| 2.5 | 2.4 | 2.20 | test of propagated `Find` error |
| 2.6 | 0.1.8 | 2.16, 2.20 | tests of `alg`/Bearer/claims |
| 2.7 | 0.1.17 | — | test of JSON without `password` |
| 2.8 | 0.1.17 | — | test of per-environment CORS |
| 2.9 | 2.16 | — | test of features disabled in prod |
| 2.10 | 0.1.17 | — | test of log redaction |
| 2.11 | 0.1.17 | 2.13 | test of timeouts and body limits |
| 2.12 | 0.1.17 | — | test of security headers |
| 2.13 | 2.11 | — | rate-limit test (429) |
| 2.14 | 0.1.5 | 3.1 | `grep` without `log.Fatal` in libraries |
| 2.15 | 0.1.17 | — | test of bcrypt/Argon2id |
| 2.16 | 2.6 | 2.9, 2.20 | test of invalid configuration |
| 2.17 | 1.5 | 2.18 | inspection of `CMD` without migration |
| 2.18 | 2.17 | — | smoke test of `entrypoint.sh` |
| 2.19 | 0.3.8 | 4.5 | `govulncheck`/`gosec`/`gitleaks` |
| 2.20 | 2.1–2.6, 0.4.2 | 3.7, 4.4 | security `go test ./...` |

### Checklist

- [x] **2.1** Validate `order_type` (`asc|desc`), each filter's `Op` and the
      column whitelist at `pkg/filters/filters.go:114`.
- [x] **2.2** Use `uuid.Parse` at `pkg/filters/filters.go:37` (it currently
      uses `uuid.FromBytes` with bytes of arbitrary length).
- [x] **2.3** Return a validation error from filters instead of logging and
      silently dropping the filter.
- [x] **2.4** Apply pagination limits: `page >= 1`, `size >= 1` and a
      configurable maximum size (`maxSize`); reject negative or huge values.
- [x] **2.5** Propagate the `Find` error at
      `pkg/pagination/pagination.go:70` instead of ignoring it.
- [x] **2.6** JWT: validate `alg` (`jwt.WithValidMethods`), require `Bearer`,
      add `iat/iss/aud/jti`, configurable expiration and leeway.
- [x] **2.7** `Password json:"-"` at `internal/core/models/user.go:12`.
- [x] **2.8** Fix CORS (`AllowOrigins: "*"` + `AllowCredentials: true`) at
      `internal/server/router.go:31`; use a per-environment whitelist.
- [x] **2.9** Disable Swagger, Asynqmon and pprof by default in production;
      Asynqmon credentials without `dev/dev` and fail startup if empty.
- [x] **2.10** Move `RequestLogger` to `slog`, redact
      body/cookies/`password`/tokens and add `X-Request-ID`.
- [x] **2.11** Add HTTP limits: `ReadHeaderTimeout`, `ReadTimeout`,
      `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes`, `http.MaxBytesReader`,
      a multipart body limit and explicit `TrustedProxies`.
- [x] **2.12** Add security headers, and secure cookie flags if sessions are
      added.
- [x] **2.13** Add rate limiting to login/sensitive endpoints.
- [x] **2.14** Remove `log.Fatal` from libraries (`pkg/converter`,
      `configs/*`); return errors and fail in `main`.
- [x] **2.15** Deprecate MD5 in `pkg/cryptography/hash.go` and use
      bcrypt/Argon2id for passwords. *(decision 4: `HashMD5` was removed and
      `HashSHA256` now returns hex; passwords remain bcrypt)*.
- [x] **2.16** Validate required configuration at startup (`JWT_SECRET` with a
      minimum length, DB/Redis credentials, `BASIC_AUTH_*` if Asynqmon is
      enabled).
- [x] **2.17** Do not migrate automatically on every startup: separate
      job/flag, no `atlas migrate apply` in the main `CMD`.
- [x] **2.18** Add `sslmode` to `DATABASE_URL` in `entrypoint.sh`, escape
      credentials with `url.URL` and use `set -eu`. *(decision 2: the URL is
      built in `cmd/migrate` with `net/url`, and `entrypoint.sh` ends up as
      `set -eu` + `exec "$@"`)*.
- [x] **2.19** Add `govulncheck`, `gosec`, `trivy` and secret scanning
      (`gitleaks`) to CI, pinned by version and publishing SARIF where
      possible.
- [x] **2.20** Add tests for filters, pagination, JWT and config validation,
      following the strategy from 0.4.

Acceptance criteria: security review with no critical findings,
`govulncheck`/`gosec`/`gitleaks` green and tests for filters/JWT/pagination.

#### Phase 2 evidence

- `REQUIRE_GOVULNCHECK=1 REQUIRE_SECURITY=1 make test-template`: **60
  passed, 1 skipped** (`compose` smoke in `minimal`). The 8-fixture matrix
  builds, runs `go vet`, `go test ./...`, `golangci-lint v2.15.0` and
  `govulncheck` cleanly.
- `gosec` (v2.30.0) green on the `full` fixture. The only real signal (G204 in
  `cmd/migrate`) was resolved with a justified `#nosec`: argv is fixed and
  `dsn` is built with `net/url`, with no shell involved. The previously pinned
  version (v2.22.9) does not support the Go 1.27.3 toolchain.
- `gitleaks` (v8.28.0) with `.gitleaks.toml`: 0 leaks on the template and on
  the generated project. Verified that the configuration still detects a
  planted token, i.e. the allowlist does not disable the tool.
- `trivy` (v0.76.1): 0 secrets in the template and the generated project.
  Vulnerability scanning stays informational (`--exit-code 0`) because it
  flags module-level CVEs that `govulncheck`, via the call graph, deems
  unreachable; both publish SARIF in CI.
- Generation fix: cookiecutter **does** render
  `{{cookiecutter.project_slug}}/.github/workflows/ci.yml`, so GitHub Actions
  expressions must be wrapped in `{% raw %}`; without that, generation failed
  with `'github' is undefined`. The root `.github/workflows/ci.yml` is not
  part of the template and is not rendered.
- `configs.Validate()` no longer reads the global `Env` when validating a copy
  of the value (`splitOrigins`), which allowed a CORS validation with `prod`
  to pass silently.
- `RequestLogger` only buffers the body when
  `ENABLE_LOG_REQUEST_DETAIL=true`; with the flag off it leaves
  `http.MaxBytesReader` intact instead of neutralizing it.
- `HashMD5` was removed (decision 4) and `HashSHA256` now returns hex.
- New tests: `internal/middlewares/{security,request_logger,ratelimit,jwt_auth}_test.go`,
  `internal/server/{router,server}_test.go`, `pkg/cryptography/hash_test.go`,
  `internal/core/models/user_test.go`, `cmd/migrate/main_test.go`,
  `configs/prod_env_test.go`, plus the extensions to `filters`, `pagination`,
  `auth` and `configs`.
- Harness: `test_generated_security_hardening` (CMD without migration, real
  smoke of `entrypoint.sh`, `security` job in CI, prod without exposed flags,
  no `log.Fatal` in `pkg/`+`configs/`+`internal/`, `Password json:"-"`),
  `test_generated_project_gosec` and `test_generated_project_gitleaks`.
- README: new "Security" section, `cmd/migrate` in the structure and
  `make security`.

## Phase 3 — Architecture

### Dependencies

| Step | Depends on | Unlocks | Evidence |
|---|---|---|---|
| 3.1 | 0.1.4, 0.1.5, 1.2 | 3.2–3.8 | http/worker startup with fx |
| 3.2 | 3.1 | — | `grep` without `GetDB`/`GetRedis`/`GetLogger` |
| 3.3 | 3.1 | 3.7 | test of config from another cwd |
| 3.4 | 3.1 | — | test of `User` with `uuid.UUID` |
| 3.5 | 3.1 | — | shutdown test with SIGTERM |
| 3.6 | 3.1 | — | test of `ListenAndServe` error |
| 3.7 | 3.1, 2.6 | — | test of the `user` flow with `JwtAuth` |
| 3.8 | 3.1, 0.3.0 | 4.5 | `curl /healthz` and `/readyz` |

### Checklist

- [x] **3.1** Move `DB`, `Redis`, `Asynq` and `Logger` from global singletons
      to fx providers with `fx.Hook` for shutdown, in both `cmd/http` and
      `cmd/worker`. *(decision 1: Viper stays, with an isolated
      `viper.New()` + `BindEnv` instance instead of Koanf; decision 2:
      `*configs.Config` is injected whole, no getters)*.
- [x] **3.2** Remove the `GetDB/GetRedis/GetLogger/GetAsynqClient` globals and
      their `log.Fatal`s. *(`configs/hooks.go` disappears: `NewConfig`,
      `NewLogger`, `NewDB`, `NewRedis` and `NewAsynqClient` return values)*.
- [x] **3.3** Replace the `runtime.Caller`-based env loading with portable
      configuration (koanf/envconfig or `viper.BindEnv`) that works from any
      directory and in tests. *(`Load(LoadOptions)` with `ENV_FILE`/`ENV_DIR`
      and upward search for `env/<stage>.env` from the cwd and from the
      binary; no `init()` or globals)*.
- [x] **3.4** Unify the PK type: `User` must use `uuid.UUID` like the
      repository/VO. *(decision 3: the initial migration
      `migrations/20261024000000_users.sql` with its `atlas.sum` is also
      included)*.
- [x] **3.5** Worker: configurable concurrency and graceful shutdown on
      SIGTERM. *(decision 4: `WORKER_CONCURRENCY=10` by default, exposed in
      `env/*.env`; `OnStop` drains tasks and respects the fx `ctx`)*.
- [x] **3.6** Fix `server.Run` to propagate `ListenAndServe` errors and shut
      down cleanly. *(`Server.Start` binds the port before starting, `Serve`
      normalizes `http.ErrServerClosed`, and `Shutdown` closes with context; a
      `Serve` failure stops the application with code ≠0)*.
- [x] **3.7** Complete the example flow (bind, validation, service, errors) in
      `user_controller.go` and register `JwtAuth`. *(`POST /signup`,
      `POST /login` and `GET /me`, with `Leeway` and rejection of
      `uuid.Nil`)*.
- [x] **3.8** Add health/readiness with real DB and Redis checks.
      *(`GET /healthz` for liveness and `GET /readyz` with conditional checks
      and a 2 s timeout)*.

Acceptance criteria: no mutable global state, testable, no `log.Fatal` in
libraries, and clean shutdown.

#### Phase 3 evidence

- `make test-template`: **68 passed, 1 skipped** (the `compose config` smoke
  skips because the local provider is `podman-compose`). The 8-fixture matrix
  passes `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint`,
  `govulncheck` and `gosec` with no findings.
- **3.1/3.2**: `configs/env.go`, `logging.go`, `client_db.go`,
  `client_redis.go` and `client_asynq.go` move to `NewX(cfg)` +
  `ProvideX(lc, ...)`; `configs/hooks.go` (with `Bootstrap`, `MustBootstrap`,
  `Shutdown` and `GetDB`) was removed. `cmd/http` registers `Start` in
  `OnStart` (the bind occupies the port before reporting startup), `Serve` in
  a goroutine that stops the app with code ≠0, and `Shutdown` in `OnStop`;
  `gin.SetMode` lives in `NewRouter`, which now returns
  `(*gin.Engine, error)`.
- **3.3**: `Load(LoadOptions)` reads `ENV_FILE`/`ENV_DIR` or searches for
  `env/<stage>.env upward from the cwd and from the binary, uses an isolated
  Viper instance with `SetDefault`+`BindEnv` via reflection, and does not
  touch package state. 7 new tests cover a foreign cwd, variable override,
  explicit `ENV_FILE`, missing file and invalid stage.
- **3.4**: `User.ID` is `uuid.UUID` (`default:gen_random_uuid()`), the VO uses
  the same key and there is a test with `reflect` over the type and tags. The
  initial migration was generated from a real `AutoMigrate` against Postgres
  16 (the DDL was dumped with `pg_dump`) and validated by applying it with
  `arigaio/atlas:1.3.5-alpine`; its `atlas.sum` was computed with
  `migrate hash`. `compose.yaml` adds the `pgdata` volume because with
  `podman-compose` the `compose run` recreates the project and lost data.
- **3.5**: `WORKER_CONCURRENCY` is in `env/{dev,testing,prod,_template}.env`,
  validated in `Validate()` and applied to `asynq.Config.Concurrency`;
  `runWorker` drains with `srv.Shutdown()` in a goroutine and returns
  `ctx.Err()` if fx times out. `cmd/worker/main_test.go` covers handler
  registration, startup and shutdown bounded by the `ctx` (with Redis
  available it runs the full cycle; without it, it only checks
  registration). The signal is covered by the smoke: `SIGTERM` ⇒ exit 0.
- **3.6**: `Server` with `Start`/`Serve`/`Shutdown`/`Addr`, full timeouts and
  tests for a busy port and for shutdown (`Serve` returns `nil` after a clean
  `Shutdown`).
- **3.7**: `POST /signup`, `POST /login` and `GET /me` with `JwtAuth`; service
  tests (bcrypt hash, credentials, nonexistent user, duplicate email) and
  controller tests (400 with detail, 201 without leaking the password, full
  flow with token, 401 with no/invalid token).
- **3.8**: `/healthz` and `/readyz` registered in `NewRouter` (exempt from
  CORS, rate limit and access log); `Readiness` adds `db` and `redis`
  depending on the flags, with a 2 s timeout and a `503` response naming the
  downed dependency. `health_test.go` covers the no-dependencies case, the
  failure and the timeout.
- New harness: `test_generated_project_architecture` verifies that no `.go`
  file contains `configs.Env`,
  `configs.GetDB/GetRedis/GetLogger/GetAsynqClient`, `MustBootstrap` nor
  `runtime.Caller`, that all `env/*.env` define `WORKER_CONCURRENCY`, that
  the worker uses `cfg.WorkerConcurrency` and that the router publishes
  `/healthz` and `/readyz`. The extended smoke applies the migration before
  starting and checks `/readyz`, the `signup → login → me` flow, and a 0 exit
  after `SIGTERM`.
- Follow-up fixes: `configs/client_redis.go` is kept with `use_asynq=yes`
  (without Redis there is no readiness or queue client) and
  `post_gen_project.py` + `test_generated_project_conditional_files` reflect
  that rule; two justified `#nosec` (`G703` in the env-search `os.Stat` and
  `G102` in the server bind to `:<port>`).
- Generators: `repository.j2` and `service.j2` no longer use `configs.GetDB`
  nor build dependencies internally; they follow the
  `NewUserRepository(db)`/`NewUserService(repo)` pattern and the project
  keeps building after `inv gen-model`.
- README: new "Architecture" section and endpoints (`/healthz`, `/readyz` and
  the JWT flow when there is a database), `WORKER_CONCURRENCY` in
  configuration, and correcting `configs.Bootstrap()` to
  `configs.NewConfig()` in the root.

## Phase 4 — DX and CI

### Dependencies

| Step | Depends on | Unlocks | Evidence |
|---|---|---|---|
| 4.1 | 0.3.2, 0.4.8 | 4.5 | `make help` and green targets |
| 4.2 | 0.3.0 | 4.5 | `atlas.hcl` parameterized per environment |
| 4.3 | 0.1.17 | 4.5 | `golangci-lint run` |
| 4.4 | 0.4.2–0.4.10 | 4.5 | coverage of the 0.4 strategy |
| 4.5 | 0.3.8, 4.1–4.4 | 4.6, 4.7 | full CI green |
| 4.6 | 4.5 | 5.7 | cache of `UV_CACHE_DIR`/`GOMODCACHE` |
| 4.7 | 4.5 | — | workflows with `concurrency`/`permissions`/`timeout` |
| 4.8 | 0.3.8 | 4.5 | `grep` without `pip install cookiecutter` |

### Checklist

- [x] **4.1** Clean up the Makefile: `lint`, `test`, `test-unit`,
      `test-integration`, `coverage`, `build-all`, `migrate`, `worker` and
      `test-template` targets.
- [x] **4.2** Parameterize `atlas.hcl` and the Makefile DSNs per environment.
- [x] **4.3** Add `.golangci.yml` and fix the warnings.
- [x] **4.4** Verify that the tests described in 0.4 are implemented and
      covered (handler with `httptest`, repository with testcontainers or
      compose); do not duplicate their definition.
- [x] **4.5** Extend the template repository's `.github/workflows/ci.yml`:
  - fixture matrix aligned with Phase 0 (minimal and full on PR; full matrix
    on nightly);
  - `uvx --with pytest --with pyyaml --with pexpect pytest -q tests/`;
  - `go build ./...`, `go vet ./...`, `go test -race ./...`,
    `go mod tidy -diff`;
  - `golangci-lint`, `govulncheck`, `gosec` and `gitleaks`;
  - `docker build` and scan of the generated project's image.
- [x] **4.6** Use `astral-sh/setup-uv` and cache `UV_CACHE_DIR` and
      `GOMODCACHE`; pin the cookiecutter and tool versions.
- [x] **4.7** Add `concurrency`, least-privilege `permissions` and `timeout`
      to the workflows.
- [x] **4.8** Do not use `pip install cookiecutter` in any workflow.

Acceptance criteria: CI green over the matrix of generated projects and
`make test-template` working from a clean state.

#### Phase 4 evidence

- Gate: `REQUIRE_GOVULNCHECK=1 REQUIRE_SECURITY=1 make test-template` with the
  full matrix → **77 passed, 1 skipped** (8:17 min); with
  `FIXTURES=minimal,full` → **29 passed, 1 skipped**. `make security` green
  (gitleaks: *no leaks*; `trivy not available` locally, tolerated by the
  target) and `git diff --check` free of issues.
- **4.1**: the root Makefile exposes `help` (with the pinned versions),
  `test-template` and `security`; the generated Makefile covers `fmt`, `vet`,
  `lint`, `build`, `build-all`, `run`, `test`, `test-unit`,
  `test-integration`, `coverage`, `migrate`, `dbdiff`, `dbapply` and
  `worker`. Harness: `test_root_makefile_help` (real labels from
  `make help`) and `test_generated_makefile_targets` (with `make -n` per
  target, `STAGE=testing go run ./cmd/migrate`, `/<slug>_test` in
  `test-integration` and `--var dev_url=` in `dbdiff`).
- **4.2**: the generated Makefile does `-include env/$(STAGE).env` (default
  `dev`), with `?=` fallbacks, `TEST_DB_NAME ?= {{ slug }}_test`,
  `TEST_DATABASE_URL` and `ATLAS_DEV_URL` (default
  `docker://postgres/16/dev?search_path=public`); `atlas.hcl:5` reads
  `var "dev_url"` and `make dbdiff` passes `--var dev_url=`. The generated
  README documents `make lint`/`build-all`/`migrate` and no longer carries a
  hardcoded DSN in "Migrations".
- **4.3**: `.golangci.yml` (standard + `bodyclose`, `errorlint`, `gocritic`,
  `misspell`, `nilerr`, `revive`, `rowserrcheck`, `sqlclosecheck`, `unparam`,
  `whitespace`, `gosec`, `gofmt`/`goimports`, exclusions for `_test.go` and
  `tools/`) with `max-issues-per-linter/same-issues: 0` so output is not
  truncated. The initial **123 warnings** (122 `revive` + 1 `gofmt`) drop to
  0: package and all-exported comments, `_ context.Context` in hooks and
  contextless handlers, `FTypeUuid → FTypeUUID` (with its tests) and `gofmt`
  in `converter_test.go` (no `{{` Jinja: an intermediate variable is used).
  The **8 fixtures** pass `gofmt -l` and a clean `golangci-lint run`.
- **4.4**: `test_full_project_http_smoke` adds `_integration_env(project)`
  (reads `env/testing.env`, uses `project.name` as the database) and runs
  `go test -tags=integration ./...` inside the same `compose up`, reusing
  the already-applied migrations.
- **4.5**: ci.yml with `on: push/pull_request` (`FIXTURES=minimal,full`),
  `schedule` (cron `17 3 * * *`, matrix of all 8) and `workflow_dispatch`
  with a `fixtures` input; a "Select fixture matrix" step via `$GITHUB_ENV`
  and an `image` job that generates the `full` fixture, runs `docker build`
  and scans with `trivy image` in informational mode (`--exit-code 0` +
  SARIF).
- **4.6/4.7**: `astral-sh/setup-uv` with `enable-cache` and
  `cache-dependency-glob`, `setup-go` with module cache (no duplicated
  `actions/cache`), workflow-level least-privilege `permissions`, workflow
  `concurrency` (`ci-{event}-{ref}`), `timeout-minutes` per job and three
  SARIF uploads (`gitleaks`, `trivy-secret`, `trivy-vuln`). In the generated
  workflow, `concurrency` is also workflow-level and no job-level block
  remains.
- **4.8**: `grep -rn "pip install cookiecutter"` with no results outside
  `PLAN.md`; `.github/workflows/ci.yml` validated with PyYAML.

## Phase 5 — Template

### Dependencies

| Step | Depends on | Unlocks | Evidence |
|---|---|---|---|
| 5.1 | 0.2.2 | 5.3 | conditional fixtures |
| 5.2 | 0.2.1 | — | `grep` of `version` in the template |
| 5.3 | 0.2.1, 5.1 | — | rendered generators |
| 5.4 | 0.2.10 | — | `env/_template.env` without broken escaping |
| 5.5 | 0.2.1 | — | correct `LICENSE` per option |
| 5.6 | 0.2.12 | — | root README reviewed |
| 5.7 | 1.2, 4.6 | — | valid `renovate.json` |
| 5.8 | 0.3.1, 0.3.3, 0.3.8 | — | `pexpect` + `--no-input` |

### Checklist

- [x] **5.1** Explicitly define which files are removed with each flag and
      test it over the fixture matrix (the flags are already defined in
      0.2).
- [x] **5.2** Use or remove the `version` variable.
- [x] **5.3** Fix the `model.j2`, `repository.j2`, `service.j2` generators to
      respect the existing pattern and DI.
- [x] **5.4** Review `_copy_without_render` and the Jinja escaping in
      `env/_template.env`.
- [x] **5.5** Generate the project's `LICENSE` according to the chosen
      license.
- [x] **5.6** Update the root README with requirements, optional features and
      the generation flow using `uvx`.
- [x] **5.7** Add `renovate.json` for Go, Docker, GitHub Actions, CI tools and
      the cookiecutter version, grouping related updates.
- [x] **5.8** Test interactive mode (smoke with `pexpect`, using
      `uvx --with pexpect`) and `--no-input` with all fixtures.

Acceptance criteria: `uvx cookiecutter` in interactive mode and `--no-input`
generate valid projects with and without optional features, and the full
matrix passes `make test-template`.

#### Phase 5 evidence

- **5.1**: `hooks/post_gen_project.py` centralizes removals in a `REMOVALS`
  table with booleans derived from the flags. The harness checks `cmd/migrate`,
  `cmd/devcli`, `tools/go.mod`/`go.sum`, the `tools/fake*` scripts,
  `atlas.hcl`, `migrations/`, the Redis/Asynq clients and `cmd/worker`; it
  also validates the `compose.yaml` services, the generated workflow's jobs
  and services, the presence/absence of conditional Makefile targets, and
  that no operational file points at a removed path.
- **5.2**: `version` was removed from `cookiecutter.json` and the eight
  fixtures (it was consumed by no artifact);
  `test_cookiecutter_context_has_no_unused_version` verifies it.
- **5.3**: `tasks.py` splits acronyms (`HTTPClient` → `http_client` /
  `httpClient`), validates the model is PascalCase, writes with a trailing
  newline and runs `gofmt`. The templates document exported symbols and the
  service exposes `GetAll`. The harness generates `HarnessModel` and
  `HTTPClient`, checks names/comments, a clean `gofmt -l` and
  `golangci-lint` after generation.
- **5.4**: `_copy_without_render` is limited to `atlas.hcl`,
  `tools/templates/*.j2` and `env/_template.env`; the harness requires that
  those files be copied byte for byte, that `atlas.hcl` keeps its Atlas
  Jinja, that the templates keep `{{ model }}` and that `env/_template.env`
  keeps `{{ STAGE }}`. The scan for remaining Jinja ignores binaries and
  covers all fixtures.
- **5.5**: `LICENSE` is rendered in four branches (MIT, Apache-2.0,
  BSD-3-Clause and proprietary) with author and UTC year. Fixtures cover all
  four licenses; the tests check markers from each text, the absence of the
  repository's original license, and that the generated README mentions the
  chosen license.
- **5.6**: root README rewritten with requirements, interactive and
  `--no-input` usage, a variables table, optional features, structure,
  license, Renovate and commands. `make cookiecutter-version` prints the pin;
  the README and the generated project's README no longer duplicate
  `cookiecutter==X.Y.Z` (test `test_root_makefile_cookiecutter_version`).
- **5.7**: `renovate.json` uses `config:recommended` and regex managers with
  `managerFilePatterns` to update Cookiecutter (PyPI), the Go version,
  `golangci-lint`, `govulncheck`, `gosec`, `gitleaks`, `trivy` and the
  templated `compose.yaml` images; official managers cover `go.mod`,
  `Dockerfile` and GitHub Actions. Tests validate the structure and that
  each regex matches the real Makefile and CI lines.
- **5.8**: `test_interactive_generation` drives the real CLI with `pexpect`
  (including the ANSI codes of the license selector), picks Apache-2.0,
  disables worker/Swagger/pprof/Sentry and builds, passes `go vet` and runs
  `go test`; `test_no_input_generation_without_config` generates with the
  defaults (`projectname`, module `github.com/yourname/projectname`, MIT and
  all features) and builds, passes `go vet` and runs `go test`.
- **Gate**: `REQUIRE_GOVULNCHECK=1 REQUIRE_SECURITY=1 make test-template` with
  the full matrix → **149 passed, 1 skipped** (7:50 min). The only skip is
  `test_generated_compose_is_valid[minimal]`, because that fixture does not
  generate `compose.yaml`; the mandatory tools (`govulncheck`, `gosec`,
  `gitleaks`) run, they are not skipped.
- **Security**: `make security` → gitleaks *no leaks found* and trivy
  secret/vulnerability with no findings.

## Notes

- Recommended order: Phase 0 (0.1 → 0.2 → 0.4.1–0.4.7 → 0.3 → 0.4.8–0.4.10) →
  1 → 2 → 3 → 4 → 5. Pulling 0.4.1–0.4.7 forward satisfies the
  `0.3.4 → 0.4.6` dependency (the `integration` build tag).
- Phase 0 and 1 can be delivered together as the first iteration.
- Each phase must end with `make test-template`; it is the only way to avoid
  regressions like the current ones.
- All tests that generate the template must use `uvx`, never a global
  cookiecutter installation.

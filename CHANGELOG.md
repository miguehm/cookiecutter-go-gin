# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
once it starts tagging releases.

## [Unreleased]

### Fixed

- The HTTP rate limiter stops its background cleanup goroutine on application
  shutdown instead of leaking it (`RateLimiter.Stop`, wired to the fx lifecycle).
- The generated CI integration job starts its Redis service with
  `--requirepass dev`, matching the `REDIS_PASS` expected by `env/testing.env`.
- The error handler no longer checks an undefined `ErrSessionInvalid` error code.
- A misconfigured `TRUSTED_PROXIES` list is logged as a warning instead of
  being silently ignored.
- `devcli fakedb` seeds three demo users with a bcrypt-hashed password instead
  of being a placeholder command.
- `Publisher.Publish` validates the Asynq client before building the task.

### Changed

- The paginated response envelope includes the computed `total` field.
- `configs.SplitList` is the single comma-separated list parser, used for both
  CORS origins and trusted proxies (previously duplicated in two packages).
- `Retry-After` on 429 responses is estimated from the token bucket delay.
- The Python test harness and template tooling are linted with Ruff
  (`make lint-py`, config in `pyproject.toml`); related findings fixed in
  `tasks.py` and `tools/fake_db.py`.

### Added

- Unit tests for previously uncovered packages: async task publishing and
  logging middleware, task handlers, JSON response envelope, API error
  sentinels, error handler, request IDs and HMAC helpers.

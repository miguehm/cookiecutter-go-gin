# Security Policy

This template takes security seriously both for itself and for the projects it
generates. Thank you for helping keep them safe.

## Supported versions

The `main` branch is the only supported version. When you generate a project
with this template, pin the Cookiecutter version (currently `2.7.1` in the
`Makefile`) so your output is reproducible.

## Reporting a vulnerability

If you find a vulnerability or a security concern in this template:

1. Do not open a public issue with the details.
2. Email the maintainers or use a GitHub security advisory
   (`Security` → `Advisories` → `New draft security advisory`) if the repo is on
   GitHub.
3. Include a description, the affected files/lines, reproduction steps and your
   assessment of the impact.

You will get an acknowledgment within 7 days. We will keep you updated while a
fix is in progress, and we will credit reporters in the changelog unless they
ask otherwise.

Please do the same for vulnerabilities discovered in your generated projects:
they are your code now, so follow your own incident process.

## Security measures already present

- Dev and example secrets are placeholder-only (`change-me`, `dev/dev`); blank
  production values fail validation at boot.
- JWTs are HS256-only, require `exp`, validate `iss`/`aud`/`nbf`/`jti` with
  leeway, and reject nil-UUID subjects.
- Login responds identically for unknown users and wrong passwords.
- SQL filters are allowlisted, DSNs are built with `net/url`, secrets are
  redacted in logs.
- The generated Dockerfile runs as a non-root user with digest-pinned images.
- `gosec`, `govulncheck` and `gitleaks` run in the template's CI and over the
  generated projects in the harness.

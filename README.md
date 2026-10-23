# cookiecutter-go-gin

Plantilla de [Cookiecutter](https://github.com/cookiecutter/cookiecutter) para
crear proyectos Go con Gin, patrón repositorio + servicio y las utilidades que
suelen hacer falta en un backend pequeño o mediano.

## Requisitos

- [uv](https://docs.astral.sh/uv/) (incluye `uvx`).
- Go 1.22 o superior.
- Docker y Docker Compose para el arnés de pruebas y los servicios locales.
- Opcional: Atlas, `golangci-lint`, `govulncheck`, `gosec`, `trivy` y
  `gitleaks`.

No instales Cookiecutter con `pip`: la versión se fija en el `Makefile` y se
ejecuta siempre con `uvx`.

## Uso

```sh
uvx --from "cookiecutter==2.7.1" cookiecutter \
  https://github.com/cuongnb14/cookiecutter-go-gin.git
```

Para generar sin preguntas:

```sh
uvx --from "cookiecutter==2.7.1" cookiecutter --no-input \
  --config-file tests/fixtures/full.yaml \
  --output-dir /tmp \
  .
```

## Variables

| Variable | Descripción |
|---|---|
| `project_name` | Nombre visible del proyecto. |
| `project_slug` | Nombre del directorio y de la base de datos. |
| `description` | Descripción corta. |
| `author`, `email` | Autor y correo. |
| `go_module` | Módulo Go explícito, por ejemplo `github.com/acme/miapp`. |
| `license` | `MIT`, `Apache-2.0`, `BSD-3-Clause` o `proprietary`. |
| `version` | Versión inicial del proyecto. |
| `use_db` | Incluye GORM, Postgres, Atlas, `cmd/atlas_loader` y `migrations/`. |
| `use_redis` | Incluye el cliente de Redis. |
| `use_asynq` | Incluye Asynq, Asynqmon y las tareas. |
| `use_worker` | Incluye `cmd/worker`. Requiere `use_asynq=yes`. |
| `use_swagger` | Incluye Swagger en `/docs`. |
| `use_pprof` | Incluye pprof. |
| `use_sentry` | Incluye Sentry en logs y middleware. |

Los flags se definen con `yes`/`no` y se validan en
`hooks/pre_gen_project.py`.

## Estructura generada

```text
cmd/http/          API HTTP
cmd/worker/        worker de Asynq (si use_worker=yes)
cmd/devcli/        utilidades de desarrollo (si use_db=yes)
cmd/atlas_loader/  loader de esquema GORM para Atlas (si use_db=yes)
configs/           configuración y clientes
internal/          controladores, servicios, repositorios y middleware
pkg/               filtros, paginación, converters y criptografía
migrations/        migraciones de Atlas (si use_db=yes)
tools/             generadores y utilidades
tests/             hueco para tests del proyecto generado
```

## Arnés de pruebas de la plantilla

El arnés vive en `tests/` (raíz de la plantilla, no dentro del proyecto
generado) y usa `uvx` para invocar la CLI real de Cookiecutter.

```sh
make test-template
```

Internamente ejecuta, para cada fixture:

```sh
uvx --with pytest --with pyyaml --with pexpect pytest -q tests/
```

Cada fixture se genera con:

```sh
uvx --from "cookiecutter==${COOKIECUTTER_VERSION}" cookiecutter --no-input \
  --config-file tests/fixtures/<fixture>.yaml \
  --output-dir "$tmp" .
```

Y sobre el proyecto generado se ejecutan `go mod tidy`, `go mod tidy -diff`,
`go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` y
`govulncheck ./...`. El arnés puede usar red y cachear `GOMODCACHE` y
`UV_CACHE_DIR`; los hooks de generación no usan red.

`golangci-lint` y `govulncheck` deben estar en `PATH`; si faltan, sus tests se
saltan. Durante la Fase 0 las vulnerabilidades de dependencias antiguas se
registran como `xfail`; usa `REQUIRE_GOVULNCHECK=1 make test-template` para
convertirlas en fallo (previsto para la Fase 1).

## Features

- Go fx para inyección de dependencias.
- Patrón repositorio + servicio.
- Asynq y Asynqmon para tareas distribuidas.
- pprof para debugging.
- GORM y Atlas para base de datos y migraciones.
- Slog + Sentry para logging.
- Viper para configuración por entorno.
- JWT auth.
- Generadores de modelo, repositorio y servicio con `invoke` + Jinja2.

## Comandos útiles

```sh
make test-template   # genera todos los fixtures y los valida
make help            # ayuda del Makefile raíz
```

## Licencia

La plantilla se distribuye bajo la licencia del repositorio. El proyecto
generado usa la licencia elegida en `license`.

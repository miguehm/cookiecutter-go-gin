# cookiecutter-go-gin

Plantilla de [Cookiecutter](https://github.com/cookiecutter/cookiecutter) para
crear proyectos Go con Gin, inyección de dependencias con `fx`, patrón
repositorio + servicio y las utilidades que suelen hacer falta en un backend
pequeño o mediano.

## Requisitos

- [uv](https://docs.astral.sh/uv/) (incluye `uvx`).
- Go 1.27 o superior.
- Docker y Docker Compose para el arnés de pruebas y para los servicios de los
  proyectos que activan base de datos, Redis o Asynq.
- Opcional: Atlas, `golangci-lint`, `govulncheck`, `gosec`, `gitleaks` y
  `trivy`.

No instales Cookiecutter con `pip`: la versión se fija en el `Makefile` y en
la CI, y se ejecuta siempre con `uvx`. Para consultarla:

```sh
make cookiecutter-version
```

## Uso

Generación interactiva (pregunta todas las variables):

```sh
version="$(make -s cookiecutter-version)"
uvx --from "cookiecutter==$version" cookiecutter .
```

Generación sin preguntas usando uno de los fixtures del arnés:

```sh
version="$(make -s cookiecutter-version)"
uvx --from "cookiecutter==$version" cookiecutter --no-input \
  --config-file tests/fixtures/full.yaml \
  --output-dir /tmp \
  .
```

Sin `--config-file`, `--no-input` usa los valores por defecto de
`cookiecutter.json`: `project_name` «Project Name» (slug `projectname`), módulo
`github.com/yourname/projectname`, licencia MIT y todas las features activas.

## Variables

| Variable | Descripción |
|---|---|
| `project_name` | Nombre visible del proyecto. |
| `project_slug` | Nombre del directorio y de la base de datos. |
| `description` | Descripción corta. |
| `author`, `email` | Autor y correo; se usan en `LICENSE` y metadatos. |
| `go_module` | Módulo Go explícito, por ejemplo `github.com/acme/miapp`. |
| `license` | `MIT`, `Apache-2.0`, `BSD-3-Clause` o `proprietary`. |
| `use_db` | Incluye GORM, Postgres, Atlas, `cmd/migrate`, `cmd/devcli`, `tools/` y `migrations/`. |
| `use_redis` | Incluye el cliente de Redis y el servicio en `compose.yaml`. |
| `use_asynq` | Incluye Asynq, Asynqmon, las tareas y el cliente de Redis. |
| `use_worker` | Incluye `cmd/worker`. Requiere `use_asynq=yes`. |
| `use_swagger` | Incluye Swagger en `/docs`. |
| `use_pprof` | Incluye pprof en `/pprof`. |
| `use_sentry` | Incluye Sentry en logs y middleware. |

Los flags se definen con `yes`/`no` y se validan en
`hooks/pre_gen_project.py`. `hooks/post_gen_project.py` elimina los archivos
que no aplican y formatea el resultado con `gofmt`.

## Features opcionales

| Flag | Por defecto | Qué añade | Requisitos |
|---|---|---|---|
| `use_db` | `yes` | GORM + Postgres, Atlas, migraciones y utilidades de desarrollo. | Docker para el smoke test. |
| `use_redis` | `yes` | Cliente de caché y servicio Redis. | Docker para el smoke test. |
| `use_asynq` | `yes` | Cola de tareas, dashboard en `/asynqmon` y cliente de Redis. | Docker para el smoke test. |
| `use_worker` | `yes` | `cmd/worker` con concurrencia configurable. | `use_asynq=yes`. |
| `use_swagger` | `yes` | Documentación en `/docs/index.html` y `make docs`. | `swag` para regenerar. |
| `use_pprof` | `yes` | Perfiles en `/pprof` cuando `ENABLE_PPROF=true`. | — |
| `use_sentry` | `yes` | Captura de errores y logs de warning. | `SENTRY_DSN` opcional. |

## Estructura generada

```text
cmd/http/                API HTTP
cmd/worker/              worker de Asynq (si use_worker=yes)
cmd/devcli/              utilidades de desarrollo (si use_db=yes)
cmd/migrate/             aplica migraciones de Atlas (si use_db=yes)
tools/cmd/atlas_loader/  loader de esquema GORM para Atlas (si use_db=yes)
tools/templates/         generadores Jinja de modelo, repositorio y servicio
configs/                 configuración y clientes
internal/                controladores, servicios, repositorios y middleware
pkg/                     filtros, paginación, converters y criptografía
migrations/              migraciones de Atlas (si use_db=yes)
tests/                   tests del proyecto generado
```

## Licencia

El proyecto generado incluye un `LICENSE` con el texto completo de la licencia
elegida:

- `MIT`, `Apache-2.0` y `BSD-3-Clause`: texto estándar con el autor y el año de
  generación.
- `proprietary`: aviso de copyright y cláusula de confidencialidad, sin texto
  de licencia abierta.

La plantilla en sí se distribuye bajo la licencia del repositorio (ver
`LICENSE` en la raíz).

## Renovate

`renovate.json` agrupa las actualizaciones de:

- módulos Go (`go.mod` y `tools/go.mod`);
- imágenes de `Dockerfile` y `compose.yaml`;
- GitHub Actions;
- versiones de las herramientas de CI (`golangci-lint`, `govulncheck`,
  `gosec`, `gitleaks`, `trivy`);
- pin de Cookiecutter en el `Makefile`.

Las versiones viven en un único sitio por herramienta; los fixtures y la
documentación las consultan con `make cookiecutter-version` o con las variables
del `Makefile`.

## Arnés de pruebas de la plantilla

El arnés vive en `tests/` (raíz de la plantilla, no dentro del proyecto
generado) y usa `uvx` para invocar la CLI real de Cookiecutter.

```sh
make test-template
FIXTURES=minimal,full make test-template   # matriz reducida
```

Cada fixture se genera con la versión fijada:

```sh
version="$(make -s cookiecutter-version)"
uvx --from "cookiecutter==$version" cookiecutter --no-input \
  --config-file tests/fixtures/<fixture>.yaml \
  --output-dir "$tmp" .
```

Sobre el proyecto generado se ejecutan `go mod tidy -diff`, `go build ./...`,
`go vet ./...`, `go test ./...`, `go test -race ./...`, `golangci-lint run`,
`govulncheck ./...`, `gosec -quiet ./...` y `gitleaks detect`. El arnés puede
usar red y cachear `GOMODCACHE` y `UV_CACHE_DIR`; los hooks de generación no
usan red.

`golangci-lint`, `govulncheck`, `gosec` y `gitleaks` deben estar en `PATH`; si
faltan, sus tests se saltan. En CI se fijan las versiones y se activan
`REQUIRE_GOVULNCHECK=1` y `REQUIRE_SECURITY=1` para que la ausencia de una
herramienta sea un fallo, no un salto:

```sh
REQUIRE_GOVULNCHECK=1 REQUIRE_SECURITY=1 make test-template
make security   # gitleaks + trivy sobre el código de la plantilla
```

## Seguridad

Lo que la plantilla aplica por defecto en el proyecto generado:

- `configs.NewConfig()` carga y valida la configuración al arrancar (`STAGE`,
  `JWT_SECRET` con longitud mínima, credenciales de DB/Redis, `BASIC_AUTH_*` si
  Asynqmon está activo, CORS en `prod` y `WORKER_CONCURRENCY`) y devuelve error
  si algo falta; `fx` aborta el arranque. Ninguna librería usa `log.Fatal`:
  devuelven errores y los `main` deciden.
- CORS por entorno (`CORS_ALLOWED_ORIGINS`/`CORS_ALLOW_CREDENTIALS`), con
  fallo cerrado en `prod` y sin `*` combinado con credenciales.
- Cabeceras de seguridad (`X-Frame-Options`, CSP, HSTS opcional), límite de
  body con `http.MaxBytesReader`, `TrustedProxies` explícitos y timeouts
  completos del servidor.
- Rate limit en memoria (`golang.org/x/time/rate`) con `/healthz` exento.
- JWT HS256 con `alg`, `iss`, `aud`, `iat`, `jti`, expiración y leeway
  configurables vía `JWT_*`.
- Logs estructurados con `slog`, `X-Request-ID` y redacción de
  `password`/`token`/`authorization`/`cookie` en bodies y query.
- Migraciones fuera del arranque: `CMD ["./http"]` y un binario `./migrate`
  (más el job de CI `integration`) que aplica `atlas migrate apply` con la URL
  construida vía `net/url`.
- La CI generada incluye `gosec`, `govulncheck`, `gitleaks` y `trivy`
  fijados por versión y publica SARIF.

## Comandos útiles

```sh
make help                 # ayuda con las versiones fijadas
make cookiecutter-version # imprime el pin de Cookiecutter
make test-template        # genera todos los fixtures y los valida
make security             # gitleaks + trivy sobre el código de la plantilla
```

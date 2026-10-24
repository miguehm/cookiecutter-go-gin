# {{ cookiecutter.project_name }}

{{ cookiecutter.description }}

Proyecto generado con `cookiecutter-go-gin`.

## Requisitos

- Go 1.27 o superior.
{% if cookiecutter.use_db == "yes" or cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}
- Docker y Docker Compose (para Postgres/Redis).
{% endif %}
{% if cookiecutter.use_db == "yes" %}
- Atlas (solo si vas a crear o aplicar migraciones manualmente).
{% endif %}

## Arranque rápido

```sh
{% if cookiecutter.use_db == "yes" or cookiecutter.use_redis == "yes" or cookiecutter.use_asynq == "yes" %}docker compose up -d
{% endif %}go mod tidy
make run
```

La aplicación escucha en `http://localhost:8080` y expone:

- `GET /healthz`: liveness; responde 200 mientras el proceso viva.
- `GET /readyz`: readiness; comprueba DB y Redis cuando están habilitados y
  responde 503 con el nombre de la dependencia caída.
{% if cookiecutter.use_db == "yes" %}- `POST /signup`, `POST /login` y `GET /me`: flujo de ejemplo autenticado con
  JWT (`Authorization: Bearer <token>` en `/me`).
{% endif %}

## Configuración

La configuración se carga desde `env/<stage>.env`. El stage por defecto es `dev`
y se puede cambiar con `STAGE`:

```sh
STAGE=testing go run ./cmd/http
```

Archivos incluidos:

- `env/dev.env`: desarrollo local.
- `env/testing.env`: tests automáticos.
- `env/_template.env`: plantilla usada por `inv format-env`.

{% if cookiecutter.use_asynq == "yes" %}- `WORKER_CONCURRENCY`: tareas en paralelo del worker (por defecto `10`).
{% endif %}Cambia `JWT_SECRET` y las credenciales de base de datos antes de desplegar.

## Estructura

```text
cmd/http/          API HTTP
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}cmd/worker/        worker de Asynq
{% endif %}{% if cookiecutter.use_db == "yes" %}cmd/devcli/        utilidades de desarrollo (dropdb, fakedb)
tools/cmd/atlas_loader/  loader de esquema GORM para Atlas (módulo tools)
{% endif %}configs/           configuración y clientes
internal/          controladores, servicios, repositorios y middleware
pkg/               filtros, paginación, converters y criptografía
{% if cookiecutter.use_db == "yes" %}migrations/        migraciones de Atlas
{% endif %}tools/             generadores y utilidades
```

## Arquitectura

- Inyección de dependencias con `fx`: `cmd/http` construye configuración,
  logger, clientes y servidor, y registra `Start`/`Shutdown` con `fx.Hook`.
- Sin estado global: no hay singletons ni `configs.GetDB`; cada componente
  recibe lo que necesita por parámetro.
- Configuración portátil: `configs.Load` lee `env/<stage>.env` con una
  instancia aislada de Viper y funciona desde cualquier directorio y en tests.
- Apagado limpio: `SIGTERM` cierra el servidor HTTP por `Shutdown` y, si hay
  worker, drena las tareas en curso.
{% if cookiecutter.use_asynq == "yes" %}
- `cmd/worker` arranca Asynq con `WORKER_CONCURRENCY` tareas en paralelo.
{% endif %}

## Comandos

```sh
make fmt                  # go fmt
make vet                  # go vet
make build                # compila los binarios
make run                  # arranca la API
make test                 # alias de test-unit
make test-unit            # tests sin servicios externos
make test-integration     # tests con build tag integration
make coverage             # cobertura de los tests unitarios
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}make worker               # arranca el worker
{% endif %}{% if cookiecutter.use_db == "yes" %}make dbreset             # recrea la base de datos y aplica migraciones
make dbdiff               # genera una migración con Atlas
make dbapply              # aplica migraciones
{% endif %}{% if cookiecutter.use_swagger == "yes" %}make docs                 # regenera Swagger
{% endif %}
```

## Tests

La estrategia recomendada es:

- Unitarios sin servicios externos: filtros, paginación, JWT y converters.
- Handlers con `httptest` y `gin.TestMode`.
- Servicios con fakes de repositorio.
- Integración de repositorios detrás del build tag `integration`.

```sh
go test ./...
go test -race ./...
go test -tags=integration ./...
```

Los tests de integración se saltan si Docker no está disponible.

## Migraciones

{% if cookiecutter.use_db == "yes" %}
```sh
atlas migrate diff --env gorm
atlas migrate apply \
  --url "postgres://dev:dev@127.0.0.1:5432/{{ cookiecutter.project_slug }}?sslmode=disable" \
  --dir file://migrations
```
{% else %}
Este proyecto se generó sin base de datos; no incluye `migrations/` ni
`tools/cmd/atlas_loader`.
{% endif %}

## Utilidades

```sh
uvx --from invoke --with jinja2 inv gen-model <ModelName>
uvx --from invoke --with jinja2 inv format-env --stage <stage|all>
```

`gen-model` genera modelo, repositorio y servicio. `format-env` renderiza
`env/_template.env` sobre el archivo del stage indicado.

## Features opcionales

{% if cookiecutter.use_db == "yes" %}
- Base de datos: GORM + Postgres + Atlas.
{% endif %}
{% if cookiecutter.use_redis == "yes" %}
- Redis: cliente de caché.
{% endif %}
{% if cookiecutter.use_asynq == "yes" %}
- Asynq: cola de tareas y dashboard en `/asynqmon`.
{% endif %}
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}
- Worker: `cmd/worker`.
{% endif %}
{% if cookiecutter.use_swagger == "yes" %}
- Swagger: documentación en `/docs/index.html`.
{% endif %}
{% if cookiecutter.use_pprof == "yes" %}
- pprof: disponible en `/pprof` cuando `ENABLE_PPROF=true`.
{% endif %}
{% if cookiecutter.use_sentry == "yes" %}
- Sentry: errores y logs de warning.
{% endif %}

## Regenerar el proyecto

```sh
uvx --from "cookiecutter==2.7.1" cookiecutter \
  --no-input \
  --output-dir /tmp \
  /ruta/a/cookiecutter-go-gin
```

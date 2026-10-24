# Plan de actualización — cookiecutter-go-gin

Objetivo: que la plantilla genere un proyecto compilable, actualizado, seguro y
con una estrategia de test clara, y mejorar su DX. Ejecutar las fases en orden;
cada fase debe terminar con el arnés de pruebas en verde (`uvx cookiecutter` +
`go build ./...` + `go vet ./...` + `go test ./...`).

## Convenciones

- Todas las generaciones de prueba usan `uvx`; nunca `pip install cookiecutter`.
- La versión de cookiecutter se fija en un único sitio (`COOKIECUTTER_VERSION`
  en el `Makefile` y en CI) y se actualiza con Renovate. No mezclar rango
  (`>=2.6,<3`) con pin exacto.
- Comando canónico:

  ```sh
  uvx --from "cookiecutter==${COOKIECUTTER_VERSION}" cookiecutter --no-input \
    --config-file tests/fixtures/<fixture>.yaml \
    --output-dir "$tmp" .
  ```

- Los tests invocan `uvx cookiecutter` mediante `subprocess`, usando la misma
  versión fijada (`--from`); no usar `pytest-cookies`, porque ejecuta
  cookiecutter en proceso y no prueba la CLI real.
- Las dependencias de test se declaran explícitamente:

  ```sh
  uvx --with pytest --with pyyaml --with pexpect pytest -q tests/
  ```

- Los hooks solo pueden usar la stdlib de Python. Si necesitan paquetes,
  declararlos explícitamente con `uvx --with`.
- El arnés de pruebas sí puede usar red (descarga de módulos Go); cachear
  `GOMODCACHE` y `UV_CACHE_DIR`. Los hooks de generación no deben depender de
  red.
- Herramientas requeridas: Go (versión de la matriz), uv/uvx, Docker y Atlas.
  Para lint y seguridad: `golangci-lint`, `govulncheck`, `gosec` y `trivy`,
  fijados por versión en CI y Makefile.
- Las referencias `archivo:línea` son orientativas; validarlas contra el código
  antes de editar.
- Los IDs de paso (`Fase.Paso`; en la Fase 0 se añade la subfase, por ejemplo
  `0.1.4`) son estables y se usan en las tablas de dependencias.

## Seguimiento

### Estados

- `[ ]` pendiente · `[~]` en curso · `[x]` hecho y verificado · `[!]` bloqueado ·
  `[–]` descartado.

Un paso solo pasa a `[x]` cuando su evidencia está registrada en la tabla de su
subfase y el gate de la fase sigue verde. `[~]`, `[!]` y `[–]` no renderizan
checkbox en GitHub, pero son legibles en crudo.

### Progreso

| Fase | Pasos | Hechos | Bloqueados | Gate |
|---|---:|---:|---:|---|
| 0 | 49 | 49 | 0 | `make test-template` |
| 1 | 7 | 7 | 0 | `go mod tidy -diff` + build reproducible |
| 2 | 20 | 20 | 0 | `govulncheck`/`gosec`/`gitleaks` verdes |
| 3 | 8 | 8 | 0 | sin globales ni `log.Fatal` en librerías |
| 4 | 8 | 0 | 0 | CI completa sobre la matriz |
| 5 | 8 | 0 | 0 | modo interactivo + `--no-input` |
| **Total** | **100** | **84** | **0** | — |

### Dependencias entre fases

| Fase | Depende de | Desbloquea |
|---|---|---|
| 0 | — | 1, 2, 3, 4, 5 |
| 1 | 0.1, 0.3 | 2, 3, 4 |
| 2 | 1, 0.3, 0.4 | 3, 4 |
| 3 | 0.1, 1, 2 | 4 |
| 4 | 0.3, 0.4, 1, 3 | 5 |
| 5 | 0.2, 0.3 | — |

## Fase 0 — Base reproducible, compilable y arnés uvx

### 0.1 Compilación y arranque

#### Dependencias

| Paso | Depende de | Desbloquea | Evidencia |
|---|---|---|---|
| 0.1.1 | — | 0.1.2 | commit base + `go build ./...` inicial |
| 0.1.2 | 0.1.1 | 0.1.3 | `go build ./...` |
| 0.1.3 | 0.1.2 | 0.1.4, 0.1.5 | `go build ./...` sin errores |
| 0.1.4 | 0.1.3 | 0.1.5 | `Bootstrap` referenciado o eliminado |
| 0.1.5 | 0.1.3, 0.1.4 | 0.1.6, 0.1.17 | arranque determinista |
| 0.1.6 | 0.1.5 | 0.1.17 | test de logger/arranque |
| 0.1.7 | 0.1.2 | 0.1.17 | `go build ./...` |
| 0.1.8 | 0.1.2 | 0.1.17 | test JWT |
| 0.1.9 | 0.1.3 | 0.1.17 | `go vet ./...` |
| 0.1.10 | 0.1.2 | 0.1.17 | `go build ./...` |
| 0.1.11 | 0.1.2 | 0.1.17 | `make` sin errores |
| 0.1.12 | 0.1.2 | 0.1.17 | `make help` |
| 0.1.13 | 0.1.2 | 0.1.17 | render de `model.j2` |
| 0.1.14 | 0.1.2 | 0.1.17 | `go vet ./...` |
| 0.1.15 | 0.1.2 | 0.1.17 | `go test ./pkg/filters/...` |
| 0.1.16 | 0.1.2 | 0.1.17 | build o `docker build` |
| 0.1.17 | 0.1.2–0.1.16 | 1.1 | `go build ./...` + `go vet ./...` |

#### Checklist

- [x] **0.1.1** Crear un commit/base de referencia y ejecutar `go build ./...`
      sobre un proyecto generado para dejar constancia del estado inicial.
- [x] **0.1.2** Añadir a `go.mod` las dependencias importadas y ausentes:
      `go.uber.org/fx`, `github.com/gin-contrib/pprof` y
      `github.com/hibiken/asynqmon`.
- [x] **0.1.3** Corregir los errores de compilación no contemplados en la
      revisión inicial:
  - `cmd/http/main.go:20`: `configs.PreServerShutdown()` no existe; debe usar
    `configs.PreServerStop()` o el hook de fx.
  - `cmd/http/main.go:25`: `configs.AsynqClient` es privado; cerrar con
    `configs.GetAsynqClient().Close()` o registrar el cierre en un `fx.Hook`.
  - Garantizar `InitLogger()` antes de `GetLogger()`: hoy ningún `main` lo
    llama y `internal/middlewares/error_handler.go:18` lo invoca durante la
    inicialización del paquete.
- [x] **0.1.4** Añadir `configs.Bootstrap()` o eliminar las llamadas en
      `cmd/worker/main.go:16` y `cmd/devcli/main.go:15`.
- [x] **0.1.5** Inicializar `InitLogger`, `InitDBClient`, `InitRedisClient` e
      `InitAsynqClient` de forma determinista y en el orden correcto.
- [x] **0.1.6** Sustituir `var logger = configs.GetLogger()` por inicialización
      perezosa en `internal/middlewares/error_handler.go:18`.
- [x] **0.1.7** Corregir la ruta Gin `docs/*any` → `/docs/*any` en
      `internal/server/router.go:49`.
- [x] **0.1.8** Evitar el panic de `fields[1]` en
      `internal/middlewares/jwt_auth.go:23` (validar esquema `Bearer` y
      longitud).
- [x] **0.1.9** Eliminar la doble llamada a `PreServerStart()` en
      `cmd/http/main.go`.
- [x] **0.1.10** Corregir `configs.GetAsynqClient.Enqueue` →
      `configs.GetAsynqClient().Enqueue(...)` en
      `internal/core/asynctasks/common.go:40`.
- [x] **0.1.11** Corregir la línea con espacios en `Makefile:25` (usar tab).
- [x] **0.1.12** Eliminar/implementar el target `mon` (`cmd/asynqmon` no existe)
      y quitar el target `push` hardcodeado a `unstable`.
- [x] **0.1.13** Corregir `tools/templates/model.j2` (evitar `gorm.Model` + `ID
      uuid.UUID` duplicado).
- [x] **0.1.14** Corregir `pkg/pagination/pagination_test.go`: importa
      `internal/models` y `models.Agency`, que no existen, y hace fallar
      `go vet ./...`.
- [x] **0.1.15** Añadir aserciones reales a `pkg/filters/filters_test.go` (ahora
      no verifica nada).
- [x] **0.1.16** Crear `migrations/` (con `.gitkeep` o migración inicial) y el
      `go.sum` que espera el `Dockerfile`; o ajustar el `Dockerfile` si se
      decide no versionarlos.
  - **Decisión:** versionar `migrations/` y `go.sum` o ajustar el `Dockerfile`;
    registrar la opción elegida en este paso.
- [x] **0.1.17** Verificar que `go build ./...` y `go vet ./...` pasan sin
      servicios externos.

### 0.2 Configuración y plantilla

#### Dependencias

| Paso | Depende de | Desbloquea | Evidencia |
|---|---|---|---|
| 0.2.1 | — | 0.2.2, 0.2.3, 0.2.9, 0.2.12, 5.2, 5.5 | `cookiecutter.json` válido |
| 0.2.2 | 0.2.1 | 0.2.4, 0.2.5, 0.2.7, 0.2.8, 0.3.1, 0.3.5, 5.1 | fixtures condicionales |
| 0.2.3 | 0.2.1 | 0.2.4, 0.2.10 | `go_module` en proyecto generado |
| 0.2.4 | 0.2.2, 0.2.3 | 0.2.10, 0.3.6 | tests negativos de `pre_gen` |
| 0.2.5 | 0.2.2 | 0.2.6 | diff env vs `EnvConfig` |
| 0.2.6 | 0.2.5 | 0.2.12 | sin secretos versionados |
| 0.2.7 | 0.2.2 | 0.2.11 | `tasks.py` funciona |
| 0.2.8 | 0.2.7 | 0.2.11 | stages alineados |
| 0.2.9 | 0.2.1 | 0.3.3 | `go mod tidy -diff` |
| 0.2.10 | 0.2.3, 0.2.4 | 5.4 | hook mínimo en verde |
| 0.2.11 | 0.2.7, 0.2.8 | 0.2.12 | referencias README válidas |
| 0.2.12 | 0.2.1, 0.2.6, 0.2.11 | 5.6 | README raíz completo |

#### Checklist

- [x] **0.2.1** Consolidar aquí todos los cambios de `cookiecutter.json`
      (eliminar la duplicidad con la Fase 5): `go_module`, autor, email,
      licencia, `version` y flags de DB/Redis/Asynq/Worker/Swagger/pprof/Sentry.
- [x] **0.2.2** Definir los flags en esta fase, antes del arnés, para que los
      fixtures puedan probar los archivos condicionales.
- [x] **0.2.3** Usar un módulo Go explícito (`go_module`) en lugar de
      `project_slug` como module path.
- [x] **0.2.4** Añadir `hooks/pre_gen_project.py` que valide `project_slug` y
      `go_module` (sin espacios, sin segmentos vacíos, sin `..`, sin empezar
      por dígito, sin palabras reservadas de Go).
- [x] **0.2.5** Alinear `env/_template.env`, `env/dev.env` y
      `configs.EnvConfig`: faltan `JWT_SECRET`, `DB_NAME`, `REDIS_DB`,
      `REDIS_TASK_DB`, `ENABLE_LOG_REQUEST_DETAIL`, `STAGE`, etc.
- [x] **0.2.6** Dejar de versionar credenciales en `env/dev.env`; usar valores
      de desarrollo no sensibles y documentar `.env.example`.
- [x] **0.2.7** Corregir `tasks.py`: lee `env/template.env` pero el archivo es
      `env/_template.env`, y renderiza `{{ .PORT }}` con Jinja (sintaxis
      inválida). Unificar el nombre y la sintaxis de la plantilla.
- [x] **0.2.8** Alinear los stages de `tasks.py` (`dev`, `testing`, `unstable`,
      `staging`) con los archivos reales o eliminar los que no existan.
- [x] **0.2.9** Decidir la estrategia de `go.sum`: incluirlo en la plantilla y
      verificar con `go mod tidy -diff`; documentar que el arnés puede usar red
      y cachear módulos, mientras que los hooks no.
  - **Decisión:** incluir `go.sum` en la plantilla y verificarlo con
    `go mod tidy -diff`; registrar la opción elegida en este paso.
- [x] **0.2.10** Reducir `hooks/post_gen_project.py` a un hook mínimo y Python
      3: formatear con `gofmt`, validar el resultado y mostrar los siguientes
      pasos. Eliminar todo el código heredado de Cookiecutter Django.
- [x] **0.2.11** Corregir referencias del README de plantilla
      (`cmd/atlas_loader`, `env/_template.env`, `invoke`) y documentar el flujo
      con `uvx`.
- [x] **0.2.12** Actualizar el README raíz: requisitos, `uvx cookiecutter`,
      estructura, comandos generados y uso de features opcionales.

### 0.3 Arnés de pruebas y CI mínima

#### Dependencias

| Paso | Depende de | Desbloquea | Evidencia |
|---|---|---|---|
| 0.3.0 | 0.1.17 | 0.3.7, 3.8, 4.2 | `docker compose config` |
| 0.3.1 | 0.2.2 | 0.3.2, 0.3.3, 5.8 | fixtures creados |
| 0.3.2 | 0.3.1 | 0.3.3, 0.3.8, 4.1 | `make test-template` |
| 0.3.3 | 0.2.9, 0.3.1, 0.3.2 | 0.3.4, 0.3.7, 0.3.9 | matriz local verde |
| 0.3.4 | 0.3.3, 0.4.6 | 0.3.8 | `go test ./...` sin servicios |
| 0.3.5 | 0.2.2, 0.3.1 | 0.3.8 | asserts de archivos condicionales |
| 0.3.6 | 0.2.4, 0.3.1 | 0.3.8 | tests negativos del hook |
| 0.3.7 | 0.3.0, 0.3.3 | 0.3.8 | smoke de `/healthz` |
| 0.3.8 | 0.3.2–0.3.7 | 2.19, 4.5, 5.8 | CI mínima verde |
| 0.3.9 | 0.3.3 | 0.3.8 | README del arnés |

#### Checklist

- [x] **0.3.0** Añadir `compose.yaml` con Postgres, Redis y Atlas (adiós a
      `devhost`). Movido desde la antigua 4.1 para que el smoke test de 0.3.7
      no dependa de una fase posterior.
- [x] **0.3.1** Crear `tests/` en la raíz del repositorio de la plantilla (no
      dentro de `{{cookiecutter.project_slug}}/`) con `test_template.py` y
      fixtures `tests/fixtures/minimal.yaml`, `no-db.yaml`, `no-redis.yaml`,
      `no-asynq.yaml`, `no-worker.yaml`, `no-swagger.yaml`, `no-sentry.yaml` y
      `full.yaml`.
- [x] **0.3.2** Añadir un `Makefile` raíz con `COOKIECUTTER_VERSION` y
      `test-template: uvx --with pytest --with pyyaml --with pexpect pytest -q
      tests/`.
- [x] **0.3.3** La prueba debe generar cada fixture con `uvx cookiecutter`,
      localizar el directorio generado y ejecutar sobre él:
      `go mod download`, `go mod tidy -diff`, `go build ./...`,
      `go vet ./...`, `go test ./...`, `golangci-lint run` y
      `govulncheck ./...`.
- [x] **0.3.4** Ejecutar `go test ./...` sin servicios externos por defecto; los
      tests de integración van detrás de un build tag (ver 0.4.6).
- [x] **0.3.5** Comprobar que los archivos condicionales existen o no según los
      flags (`DB`, `Redis`, `Asynq`, `Worker`, `Swagger`, `pprof`, `Sentry`).
- [x] **0.3.6** Añadir pruebas negativas de `pre_gen_project.py`: slugs y
      módulos inválidos deben fallar con un mensaje claro.
- [x] **0.3.7** Añadir un smoke test de arranque cuando el fixture requiera
      DB/Redis: `compose up`, esperar healthchecks, comprobar `/healthz` y
      apagar. Debe hacer `skip` si Docker no está disponible.
- [x] **0.3.8** Añadir una CI mínima desde esta fase (ver Fase 4 para la CI
      completa): instalar Go y `astral-sh/setup-uv`, cachear `UV_CACHE_DIR` y
      `GOMODCACHE`, instalar las herramientas fijadas y ejecutar
      `uvx --with pytest --with pyyaml --with pexpect pytest -q tests/`.
- [x] **0.3.9** Documentar el arnés en el README raíz, incluyendo que la
      generación de pruebas usa `uvx`.

### 0.4 Estrategia de test del proyecto generado

#### Dependencias

| Paso | Depende de | Desbloquea | Evidencia |
|---|---|---|---|
| 0.4.1 | — | 0.4.7 | README con la pirámide de tests |
| 0.4.2 | 0.4.7 | 0.4.3, 0.4.6, 0.4.8, 2.20 | `go test ./...` unitario |
| 0.4.3 | 0.4.2 | 0.4.6 | tests de handler con `httptest` |
| 0.4.4 | 0.4.2 | 0.4.6 | tests de servicio con fakes |
| 0.4.5 | 0.4.7 | 0.4.6 | integración con `t.Skip` |
| 0.4.6 | 0.4.2–0.4.5 | 0.3.4, 0.4.8, 0.4.9, 0.4.10 | tag `integration` |
| 0.4.7 | 0.4.1 | 0.4.2, 0.4.5 | helpers en `internal/testutil` |
| 0.4.8 | 0.4.6 | 4.1, 4.4 | targets `test*` |
| 0.4.9 | 0.4.6 | 4.4, 4.5 | CI del proyecto generado |
| 0.4.10 | 0.4.6 | 2.20, 4.4 | separación unit/integration |

#### Checklist

- [x] **0.4.1** Definir en el README del proyecto generado la pirámide de tests
      y los comandos para ejecutarla.
- [x] **0.4.2** Unitarios puros (sin DB/Redis): filtros, paginación, JWT,
      validación de configuración y converters. Tabla de casos, `t.Parallel()`
      y `go test -race`.
- [x] **0.4.3** Handlers con `httptest` + `gin.TestMode`, verificando códigos,
      body y errores.
- [x] **0.4.4** Servicios con fakes de las interfaces de repositorio ya
      existentes.
- [x] **0.4.5** Repositorios solo en integración, con `testcontainers-go` o
      `compose`: esquema/DB aislado por test, `t.Cleanup` y `t.Skip` si Docker
      no está.
- [x] **0.4.6** Aislar la integración con build tag `integration` para que
      `go test ./...` pase sin servicios externos.
- [x] **0.4.7** Crear helpers en `internal/testutil` (fixtures, configuración
      inyectada, factories) y evitar depender de `devhost` o de estado global.
- [x] **0.4.8** Añadir targets `test`, `test-unit`, `test-integration` y
      `coverage` al Makefile generado.
- [x] **0.4.9** Incluir un workflow de CI en el proyecto generado con
      `go test -race ./...`, cobertura y el job de integración.
- [x] **0.4.10** Sustituir el `test-resetdb` actual (que exige DB y Atlas para
      cualquier test) por una separación explícita unit/integration.

Criterio de aceptación de la Fase 0: `make test-template` genera todos los
fixtures; cada proyecto compila, pasa `go vet`, `go test` sin servicios
externos, pasa `golangci-lint` y `govulncheck`, y arranca con Postgres/Redis
(según sus flags) superando el smoke test de `/healthz`.

#### Evidencia Fase 0

- `make test-template`: 42 pasados, 9 saltados (herramientas ausentes y
  `compose.yaml` en `minimal`) en 61 s.
- Arnés con `golangci-lint` y `govulncheck` instalados: 42 pasados, 1 saltado,
  8 `xfail`. `golangci-lint` está limpio en los 8 fixtures. `govulncheck`
  detecta vulnerabilidades de dependencias antiguas y del stdlib local; queda
  como `xfail` documentado hasta la Fase 1 (`REQUIRE_GOVULNCHECK=1` lo hace
  bloqueante).
- Smoke test `full`: Postgres y Redis de `compose.yaml`, API en `/healthz`,
  `docker compose down -v` en `finally`.
- Generación de modelo con `inv`: `HarnessModel` produce modelo, repositorio y
  servicio, y el proyecto sigue compilando.
- No se pudo ejecutar el workflow remoto de GitHub Actions desde este entorno;
  el YAML del arnés y el generado se validan con PyYAML en los tests.

## Fase 1 — Actualización

### Dependencias

| Paso | Depende de | Desbloquea | Evidencia |
|---|---|---|---|
| 1.1 | 0.1.17 | 1.2 | `go build ./...` con Go 1.27 |
| 1.2 | 1.1 | 1.3, 5.7 | `go list -m -u all` |
| 1.3 | 1.2 | 1.4, 1.6, 1.7 | `go mod tidy -diff` |
| 1.4 | 1.3 | 1.5 | decisión de imagen final registrada |
| 1.5 | 1.4 | 1.7, 2.17 | `docker build` |
| 1.6 | 1.3 | 1.7 | build de `tools/` |
| 1.7 | 1.3, 1.5, 1.6 | 2.1 | build reproducible + `go mod tidy -diff` |

### Checklist

- [x] **1.1** Subir `go` a 1.27 y la imagen base a `golang:1.27-alpine`
      (pinneada por digest). Verificar disponibilidad antes de fijar la
      versión; si no existe, usar la última estable y actualizar este plan.
  - **Ejecutado:** `go 1.27.3`, `golang:1.27-alpine@sha256:b8c9b82a…`,
    `alpine:3.22.7@sha256:2e0b81d6…`, CI raíz y CI generada alineadas.
- [x] **1.2** Actualizar dependencias verificando cada salto con
      `go list -m -u all` y `go get`; no fijar versiones a ciegas. Las
      versiones de referencia iniciales son:
  - `gin 1.10.0 → 1.12.0`
  - `gorm 1.25.12 → 1.31.2`, `driver/postgres 1.5.11 → 1.6.3`
  - `atlas-provider-gorm 0.5.0 → 0.6.1`
  - `asynq 0.25.1 → 0.26.0` + `asynqmon 0.7.2`
  - `viper 1.19.0 → 1.21.0`
  - `golang-jwt/jwt/v5 5.2.1 → 5.3.1`
  - `go-redis/v9 9.7.0 → 9.23.0`
  - `sentry-go 0.30.0 → 0.50.0`
  - `fx → 1.24.0`
  - Además: `cors 1.7.9`, `pprof 1.5.6`, `validator/v10 10.30.5`,
    `gin-swagger 1.6.1`, `slog-multi 1.8.0`, `slog-sentry/v2 2.11.0`,
    `urfave/cli/v2 2.27.7`, `x/crypto 0.58.0`, `decimal 1.6.0` y
    `sentry-go/gin 0.50.0`.
- [x] **1.3** Ejecutar `go mod tidy` y eliminar las dependencias indirectas que
      ya no se usen (drivers MySQL/SQLServer/SQLite si se separa Atlas).
  - **Ejecutado:** `go.sum` raíz y `tools/go.sum` regenerados desde el fixture
    `full`; `go mod tidy -diff` limpio en ambos módulos.
- [x] **1.4** Resolver la contradicción de imagen final: `distroless/static` no
      tiene shell, así que no puede ejecutar `entrypoint.sh`. Elegir una
      opción: mantener una base con shell, usar un binario Go de arranque, o
      mover las migraciones a un servicio/job aparte y dejar `CMD` apuntando al
      binario. Considerar `tini`/`dumb-init` como PID 1 si se mantiene un
      entrypoint.
  - **Decisión:** base con shell (`alpine:3.22.7` pinneada por digest) + `tini`
    como PID 1 y `entrypoint.sh` con migración. Distroless se reevaluará en
    2.17, cuando las migraciones pasen a un job separado.
- [x] **1.5** Dockerfile multi-stage:
  - usuario no-root;
  - imagen final mínima y pinneada por digest;
  - `-trimpath -ldflags="-s -w"`;
  - `ARG TARGETARCH` en lugar de `GOARCH=amd64` fijo;
  - Atlas pinneado con checksum;
  - `HEALTHCHECK` y build multiarch con `buildx`.
- [x] **1.6** Separar `cmd/atlas_loader` en `tools/go.mod` para no arrastrar
      drivers MySQL/SQLServer/SQLite y actualizar `atlas.hcl` en consecuencia
      (por ejemplo, `go -C tools run ./cmd/atlas_loader`). Añadir `go.work` si
      conviven ambos módulos.
  - **Decisión:** módulo anidado `{{ cookiecutter.go_module }}/tools` con
    `replace` al módulo raíz y `go -C tools`; sin `go.work`. El hook elimina
    `tools/cmd`, `tools/go.mod` y `tools/go.sum` cuando `use_db=no`.
- [x] **1.7** Verificar `go mod tidy -diff` y que el build sea reproducible.

Criterio de aceptación: `go list -m -u all` sin actualizaciones relevantes,
`go mod tidy -diff` limpio y build reproducible.

#### Evidencia Fase 1

- `make test-template` con `REQUIRE_GOVULNCHECK=1`: **50 pasados, 1 saltado**
  (smoke de `compose` en `minimal`). `golangci-lint v2.15.0` limpio en los 8
  fixtures y `govulncheck` sin vulnerabilidades explotables.
- `go mod tidy -diff` limpio en el módulo raíz y en `tools` para el fixture
  `full`; `go -C tools build ./...` y `go -C tools vet ./...` limpios.
- Build reproducible: dos compilaciones de `cmd/http` con
  `-trimpath -buildvcs=false -ldflags="-s -w"` producen el mismo SHA-256
  (`06acd74a…`).
- `go -C tools run ./cmd/atlas_loader` genera el esquema SQL de `users`.
- `docker build` del fixture `full` correcto en `linux/amd64` (Atlas 1.3.5
  verificado por checksum). El build `linux/arm64` no se pudo ejecutar en local
  por falta de emulación/binfmt; queda cubierto por la CI multiarch de la
  Fase 4.
- Se corrigió `pkg/pagination/pagination.go:63` (`Statement.DB` → `Statement`)
  para mantener `golangci-lint` en verde con `gorm 1.31.2`.
- Actualizaciones pendientes en `go list -m -u all`: solo indirectas
  transitivas; todas las dependencias directas están en su última versión.

## Fase 2 — Seguridad

### Dependencias

| Paso | Depende de | Desbloquea | Evidencia |
|---|---|---|---|
| 2.1 | 0.1.17 | 2.2, 2.3, 2.20 | tests table-driven de filtros |
| 2.2 | 2.1 | 2.20 | test de `uuid.Parse` con bytes inválidos |
| 2.3 | 2.1 | 2.20 | test de filtro inválido con error |
| 2.4 | 0.1.17 | 2.5, 2.20 | test de límites `page`/`size` |
| 2.5 | 2.4 | 2.20 | test de error de `Find` propagado |
| 2.6 | 0.1.8 | 2.16, 2.20 | tests de `alg`/Bearer/claims |
| 2.7 | 0.1.17 | — | test de JSON sin `password` |
| 2.8 | 0.1.17 | — | test de CORS por entorno |
| 2.9 | 2.16 | — | test de features desactivadas en prod |
| 2.10 | 0.1.17 | — | test de redacción de logs |
| 2.11 | 0.1.17 | 2.13 | test de timeouts y límites de body |
| 2.12 | 0.1.17 | — | test de cabeceras de seguridad |
| 2.13 | 2.11 | — | test de rate limit (429) |
| 2.14 | 0.1.5 | 3.1 | `grep` sin `log.Fatal` en librerías |
| 2.15 | 0.1.17 | — | test de bcrypt/Argon2id |
| 2.16 | 2.6 | 2.9, 2.20 | test de configuración inválida |
| 2.17 | 1.5 | 2.18 | inspección de `CMD` sin migración |
| 2.18 | 2.17 | — | smoke test de `entrypoint.sh` |
| 2.19 | 0.3.8 | 4.5 | `govulncheck`/`gosec`/`gitleaks` |
| 2.20 | 2.1–2.6, 0.4.2 | 3.7, 4.4 | `go test ./...` de seguridad |

### Checklist

- [x] **2.1** Validar `order_type` (`asc|desc`), el `Op` de cada filtro y la
      whitelist de columnas en `pkg/filters/filters.go:114`.
- [x] **2.2** Usar `uuid.Parse` en `pkg/filters/filters.go:37` (hoy usa
      `uuid.FromBytes` con bytes de longitud arbitraria).
- [x] **2.3** Devolver error de validación en filtros en lugar de loguear y
      omitir el filtro silenciosamente.
- [x] **2.4** Aplicar límites de paginación: `page >= 1`, `size >= 1` y tamaño
      máximo configurable (`maxSize`); rechazar valores negativos o gigantes.
- [x] **2.5** Propagar el error de `Find` en `pkg/pagination/pagination.go:70`
      en lugar de ignorarlo.
- [x] **2.6** JWT: validar `alg` (`jwt.WithValidMethods`), exigir `Bearer`,
      añadir `iat/iss/aud/jti`, expiración configurable y leeway.
- [x] **2.7** `Password json:"-"` en `internal/core/models/user.go:12`.
- [x] **2.8** Corregir CORS (`AllowOrigins: "*"` + `AllowCredentials: true`) en
      `internal/server/router.go:31`; usar whitelist por entorno.
- [x] **2.9** Desactivar por defecto Swagger, Asynqmon y pprof en producción;
      credenciales de Asynqmon sin `dev/dev` y fallo al arrancar si están
      vacías.
- [x] **2.10** Pasar `RequestLogger` a `slog`, redactar
      body/cookies/`password`/tokens y añadir `X-Request-ID`.
- [x] **2.11** Añadir límites HTTP: `ReadHeaderTimeout`, `ReadTimeout`,
      `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes`, `http.MaxBytesReader`,
      límite de body multipart y `TrustedProxies` explícitos.
- [x] **2.12** Añadir cabeceras de seguridad y flags seguros de cookies si se
      añaden sesiones.
- [x] **2.13** Añadir rate limit en login/endpoints sensibles.
- [x] **2.14** Quitar `log.Fatal` de librerías (`pkg/converter`, `configs/*`);
      devolver errores y fallar en `main`.
- [x] **2.15** Deprecar MD5 en `pkg/cryptography/hash.go` y usar
      bcrypt/Argon2id para contraseñas. *(decisión 4: `HashMD5` se eliminó y
      `HashSHA256` pasa a devolver hex; las contraseñas siguen en bcrypt)*.
- [x] **2.16** Validar configuración obligatoria al arrancar (`JWT_SECRET` con
      longitud mínima, credenciales de DB/Redis, `BASIC_AUTH_*` si Asynqmon
      está activo).
- [x] **2.17** No migrar automáticamente en cada arranque: job/flag separado,
      sin `atlas migrate apply` en el `CMD` principal.
- [x] **2.18** Añadir `sslmode` a `DATABASE_URL` en `entrypoint.sh`, escapar
      credenciales con `url.URL` y usar `set -eu`. *(decisión 2: la URL se
      construye en `cmd/migrate` con `net/url`, `entrypoint.sh` queda en
      `set -eu` + `exec "$@"`)*.
- [x] **2.19** Añadir `govulncheck`, `gosec`, `trivy` y escaneo de secretos
      (`gitleaks`) a la CI, fijados por versión y publicando SARIF cuando sea
      posible.
- [x] **2.20** Añadir tests para filtros, paginación, JWT y validación de
      configuración, siguiendo la estrategia de 0.4.

Criterio de aceptación: revisión de seguridad sin hallazgos críticos,
`govulncheck`/`gosec`/`gitleaks` en verde y tests de filtros/JWT/paginación.

#### Evidencia Fase 2

- `REQUIRE_GOVULNCHECK=1 REQUIRE_SECURITY=1 make test-template`: **60
  pasados, 1 saltado** (smoke de `compose` en `minimal`). La matriz de los 8
  fixtures compila, hace `go vet`, `go test ./...`, `golangci-lint v2.15.0` y
  `govulncheck` limpios.
- `gosec` (v2.30.0) en verde sobre el fixture `full`. La única señal real
  (G204 en `cmd/migrate`) se resolvió con `#nosec` justificado: el argv es
  fijo y `dsn` se construye con `net/url`, sin shell de por medio. La versión
  fijada antes (v2.22.9) no soporta el toolchain de Go 1.27.3.
- `gitleaks` (v8.28.0) con `.gitleaks.toml`: 0 fugas sobre la plantilla y
  sobre el proyecto generado. Verificado que la configuración sigue
  detectando un token plantado, es decir, que el allowlist no apaga la
  herramienta.
- `trivy` (v0.76.1): 0 secretos en la plantilla y en el generado. El escaneo
  de vulnerabilidades queda informativo (`--exit-code 0`) porque marca CVEs a
  nivel de módulo que `govulncheck`, por grafo de llamadas, considera no
  alcanzables; ambos publican SARIF en la CI.
- Corrección de generación: cookiecutter **sí** renderiza
  `{{cookiecutter.project_slug}}/.github/workflows/ci.yml`, por lo que las
  expresiones de GitHub Actions van envueltas en `{% raw %}`; sin eso la
  generación fallaba con `'github' is undefined`. El `.github/workflows/ci.yml`
  de la raíz no forma parte del template y no se renderiza.
- `configs.Validate()` ya no lee el global `Env` al validar una copia del
  valor (`splitOrigins`), lo que permitía que una validación con CORS en
  `prod` pasara en silencio.
- `RequestLogger` solo amortigua el body cuando `ENABLE_LOG_REQUEST_DETAIL=true`;
  con el flag apagado deja `http.MaxBytesReader` intacto en vez de
  neutralizarlo.
- `HashMD5` se eliminó (decisión 4) y `HashSHA256` ahora devuelve hex.
- Nuevos tests: `internal/middlewares/{security,request_logger,ratelimit,jwt_auth}_test.go`,
  `internal/server/{router,server}_test.go`, `pkg/cryptography/hash_test.go`,
  `internal/core/models/user_test.go`, `cmd/migrate/main_test.go`,
  `configs/prod_env_test.go`, más las ampliaciones de `filters`, `pagination`,
  `auth` y `configs`.
- Arnés: `test_generated_security_hardening` (CMD sin migración, smoke real
  de `entrypoint.sh`, job `security` en la CI, prod sin flags expuestos, sin
  `log.Fatal` en `pkg/`+`configs/`+`internal/`, `Password json:"-"`),
  `test_generated_project_gosec` y `test_generated_project_gitleaks`.
- README: nueva sección «Seguridad», `cmd/migrate` en la estructura y
  `make security`.

## Fase 3 — Arquitectura

### Dependencias

| Paso | Depende de | Desbloquea | Evidencia |
|---|---|---|---|
| 3.1 | 0.1.4, 0.1.5, 1.2 | 3.2–3.8 | arranque http/worker con fx |
| 3.2 | 3.1 | — | `grep` sin `GetDB`/`GetRedis`/`GetLogger` |
| 3.3 | 3.1 | 3.7 | test de configuración desde otro cwd |
| 3.4 | 3.1 | — | test de `User` con `uuid.UUID` |
| 3.5 | 3.1 | — | test de shutdown con SIGTERM |
| 3.6 | 3.1 | — | test de error de `ListenAndServe` |
| 3.7 | 3.1, 2.6 | — | test del flujo `user` con `JwtAuth` |
| 3.8 | 3.1, 0.3.0 | 4.5 | `curl /healthz` y `/readyz` |

### Checklist

- [x] **3.1** Migrar `DB`, `Redis`, `Asynq` y `Logger` de singletons globales a
      providers de fx con `fx.Hook` para cierre, tanto en `cmd/http` como en
      `cmd/worker`. *(decisión 1: se mantiene Viper con una instancia aislada
      `viper.New()` + `BindEnv` en lugar de Koanf; decisión 2: `*configs.Config`
      se inyecta completo, sin getters)*.
- [x] **3.2** Eliminar los `GetDB/GetRedis/GetLogger/GetAsynqClient` globales y
      sus `log.Fatal`. *(`configs/hooks.go` desaparece: `NewConfig`,
      `NewLogger`, `NewDB`, `NewRedis` y `NewAsynqClient` devuelven valores)*.
- [x] **3.3** Sustituir la carga de env basada en `runtime.Caller` por
      configuración portable (koanf/envconfig o `viper.BindEnv`) que funcione
      desde cualquier directorio y en tests. *(`Load(LoadOptions)` con
      `ENV_FILE`/`ENV_DIR` y búsqueda ascendente de `env/<stage>.env` desde el
      cwd y desde el binario; sin `init()` ni globals)*.
- [x] **3.4** Unificar el tipo de PK: `User` debe usar `uuid.UUID` como el
      repositorio/VO. *(decisión 3: además se incluye la migración inicial
      `migrations/20261024000000_users.sql` con su `atlas.sum`)*.
- [x] **3.5** Worker: concurrencia configurable y shutdown graceful ante
      SIGTERM. *(decisión 4: `WORKER_CONCURRENCY=10` por defecto, expuesto en
      `env/*.env`; `OnStop` drena las tareas y respeta el `ctx` de fx)*.
- [x] **3.6** Corregir `server.Run` para propagar errores de `ListenAndServe` y
      apagarse limpiamente. *(`Server.Start` reserva el puerto antes de arrancar,
      `Serve` normaliza `http.ErrServerClosed` y `Shutdown` cierra con
      contexto; el fallo de `Serve` para la aplicación con código ≠0)*.
- [x] **3.7** Completar el flujo de ejemplo (bind, validación, servicio,
      errores) en `user_controller.go` y registrar `JwtAuth`. *(`POST /signup`,
      `POST /login` y `GET /me`, con `Leeway` y rechazo de `uuid.Nil`)*.
- [x] **3.8** Añadir health/readiness con comprobación real de DB y Redis.
      *(`GET /healthz` de liveness y `GET /readyz` con comprobaciones
      condicionales y timeout de 2 s)*.

Criterio de aceptación: sin estado global mutable, testeable, sin `log.Fatal`
en librerías y con apagado limpio.

#### Evidencia Fase 3

- `make test-template`: **68 pasados, 1 saltado** (el smoke de `compose config`
  se salta porque el provider local es `podman-compose`). La matriz de los 8
  fixtures pasa `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint`, `govulncheck` y `gosec` sin hallazgos.
- **3.1/3.2**: `configs/env.go`, `logging.go`, `client_db.go`,
  `client_redis.go` y `client_asynq.go` pasan a `NewX(cfg)` +
  `ProvideX(lc, ...)`; `configs/hooks.go` (con `Bootstrap`, `MustBootstrap`,
  `Shutdown` y `GetDB`) se eliminó. `cmd/http` registra `Start` en `OnStart`
  (el bind ocupa el puerto antes de reportar arranque), `Serve` en una goroutine
  que para la app con código ≠0 y `Shutdown` en `OnStop`; `gin.SetMode` vive en
  `NewRouter`, que ahora devuelve `(*gin.Engine, error)`.
- **3.3**: `Load(LoadOptions)` lee `ENV_FILE`/`ENV_DIR` o busca
  `env/<stage>.env` hacia arriba desde el cwd y desde el binario, usa una
  instancia aislada de Viper con `SetDefault`+`BindEnv` por reflexión y no
  toca estado del paquete. 7 tests nuevos cubren cwd ajeno, override de
  variable, `ENV_FILE` explícito, archivo ausente y stage inválido.
- **3.4**: `User.ID` es `uuid.UUID` (`default:gen_random_uuid()`), el VO usa la
  misma clave y hay test con `reflect` sobre el tipo y las tags. La migración
  inicial se generó desde un `AutoMigrate` real contra Postgres 16 (se vertió
  el DDL con `pg_dump`) y se validó aplicándola con
  `arigaio/atlas:1.3.5-alpine`; su `atlas.sum` se calculó con
  `migrate hash`. `compose.yaml` añade el volumen `pgdata` porque con
  `podman-compose` el `compose run` recrea el proyecto y perdía los datos.
- **3.5**: `WORKER_CONCURRENCY` está en `env/{dev,testing,prod,_template}.env`,
  se valida en `Validate()` y se aplica en `asynq.Config.Concurrency`;
  `runWorker` drena con `srv.Shutdown()` en una goroutine y devuelve `ctx.Err()`
  si fx agota su timeout. `cmd/worker/main_test.go` cubre el registro de
  handlers, el arranque y el apagado acotado por el `ctx` (con Redis disponible
  recorre el ciclo completo; sin él solo verifica el registro). La señal la
  cubre el smoke: `SIGTERM` ⇒ código 0.
- **3.6**: `Server` con `Start`/`Serve`/`Shutdown`/`Addr`, timeouts completos y
  tests de puerto ocupado y de apagado (`Serve` devuelve `nil` tras un
  `Shutdown` limpio).
- **3.7**: `POST /signup`, `POST /login` y `GET /me` con `JwtAuth`; tests de
  servicio (hash bcrypt, credenciales, usuario inexistente, email duplicado) y
  de controlador (400 con detalle, 201 sin filtrar contraseña, flujo completo
  con token, 401 sin/ con token inválido).
- **3.8**: `/healthz` y `/readyz` registrados en `NewRouter` (exentos de CORS,
  rate limit y log de acceso); `Readiness` añade `db` y `redis` según los
  flags, con timeout de 2 s y respuesta `503` nombrando la dependencia caída.
  `health_test.go` cubre el caso sin dependencias, el fallo y el timeout.
- Arnés nuevo: `test_generated_project_architecture` verifica que ningún `.go`
  contiene `configs.Env`, `configs.GetDB/GetRedis/GetLogger/GetAsynqClient`,
  `MustBootstrap` ni `runtime.Caller`, que todos los `env/*.env` definen
  `WORKER_CONCURRENCY`, que el worker usa `cfg.WorkerConcurrency` y que el
  router publica `/healthz` y `/readyz`. El smoke ampliado aplica la migración
  antes de arrancar y comprueba `/readyz`, el flujo `signup → login → me` y la
  salida 0 tras `SIGTERM`.
- Correcciones derivadas: `configs/client_redis.go` se conserva con
  `use_asynq=yes` (sin Redis no hay readiness ni cliente de cola) y
  `post_gen_project.py` + `test_generated_project_conditional_files` reflejan
  esa regla; dos `#nosec` justificados (`G703` en `os.Stat` de la búsqueda del
  env y `G102` en el bind `:<puerto>` del servidor).
- Generadores: `repository.j2` y `service.j2` ya no usan `configs.GetDB` ni
  construyen dependencias internamente, siguen el patrón de
  `NewUserRepository(db)`/`NewUserService(repo)` y el proyecto sigue compilando
  tras `inv gen-model`.
- README: secciones «Arquitectura» y endpoints (`/healthz`, `/readyz` y el flujo
  JWT cuando hay base de datos), `WORKER_CONCURRENCY` en configuración y
  corrección de `configs.Bootstrap()` por `configs.NewConfig()` en la raíz.

## Fase 4 — DX y CI

### Dependencias

| Paso | Depende de | Desbloquea | Evidencia |
|---|---|---|---|
| 4.1 | 0.3.2, 0.4.8 | 4.5 | `make help` y targets verdes |
| 4.2 | 0.3.0 | 4.5 | `atlas.hcl` parametrizado por entorno |
| 4.3 | 0.1.17 | 4.5 | `golangci-lint run` |
| 4.4 | 0.4.2–0.4.10 | 4.5 | cobertura de la estrategia 0.4 |
| 4.5 | 0.3.8, 4.1–4.4 | 4.6, 4.7 | CI completa verde |
| 4.6 | 4.5 | 5.7 | cache de `UV_CACHE_DIR`/`GOMODCACHE` |
| 4.7 | 4.5 | — | workflows con `concurrency`/`permissions`/`timeout` |
| 4.8 | 0.3.8 | 4.5 | `grep` sin `pip install cookiecutter` |

### Checklist

- [ ] **4.1** Limpiar Makefile: targets `lint`, `test`, `test-unit`,
      `test-integration`, `coverage`, `build-all`, `migrate`, `worker` y
      `test-template`.
- [ ] **4.2** Parametrizar `atlas.hcl` y los DSN del Makefile por entorno.
- [ ] **4.3** Añadir `.golangci.yml` y arreglar los avisos.
- [ ] **4.4** Verificar que los tests descritos en 0.4 están implementados y
      cubiertos (handler con `httptest`, repositorio con testcontainers o
      compose); no duplicar su definición.
- [ ] **4.5** Ampliar `.github/workflows/ci.yml` del repositorio plantilla:
  - matriz de fixtures alineada con la Fase 0 (mínima y completa en PR; matriz
    completa en nightly);
  - `uvx --with pytest --with pyyaml --with pexpect pytest -q tests/`;
  - `go build ./...`, `go vet ./...`, `go test -race ./...`,
    `go mod tidy -diff`;
  - `golangci-lint`, `govulncheck`, `gosec` y `gitleaks`;
  - `docker build` y escaneo de la imagen del proyecto generado.
- [ ] **4.6** Usar `astral-sh/setup-uv` y cachear `UV_CACHE_DIR` y
      `GOMODCACHE`; fijar la versión de cookiecutter y de las herramientas.
- [ ] **4.7** Añadir `concurrency`, `permissions` mínimos y `timeout` a los
      workflows.
- [ ] **4.8** No usar `pip install cookiecutter` en ningún workflow.

Criterio de aceptación: CI verde sobre la matriz de proyectos generados y
`make test-template` funciona en limpio.

## Fase 5 — Plantilla

### Dependencias

| Paso | Depende de | Desbloquea | Evidencia |
|---|---|---|---|
| 5.1 | 0.2.2 | 5.3 | fixtures condicionales |
| 5.2 | 0.2.1 | — | `grep` de `version` en la plantilla |
| 5.3 | 0.2.1, 5.1 | — | generadores renderizados |
| 5.4 | 0.2.10 | — | `env/_template.env` sin escaping roto |
| 5.5 | 0.2.1 | — | `LICENSE` correcto por opción |
| 5.6 | 0.2.12 | — | README raíz revisado |
| 5.7 | 1.2, 4.6 | — | `renovate.json` válido |
| 5.8 | 0.3.1, 0.3.3, 0.3.8 | — | `pexpect` + `--no-input` |

### Checklist

- [ ] **5.1** Definir explícitamente qué archivos se eliminan con cada flag y
      probarlo en la matriz de fixtures (los flags ya se definen en 0.2).
- [ ] **5.2** Usar o eliminar la variable `version`.
- [ ] **5.3** Corregir generadores `model.j2`, `repository.j2`, `service.j2` para
      que respeten el patrón y la DI existentes.
- [ ] **5.4** Revisar `_copy_without_render` y el escapado de Jinja en
      `env/_template.env`.
- [ ] **5.5** Generar el `LICENSE` del proyecto según la licencia elegida.
- [ ] **5.6** Actualizar README raíz con requisitos, features opcionales y flujo
      de generación usando `uvx`.
- [ ] **5.7** Añadir `renovate.json` para Go, Docker, GitHub Actions,
      herramientas de CI y la versión de cookiecutter, agrupando
      actualizaciones relacionadas.
- [ ] **5.8** Probar el modo interactivo (smoke con `pexpect`, usando
      `uvx --with pexpect`) y `--no-input` con todos los fixtures.

Criterio de aceptación: `uvx cookiecutter` en modo interactivo y `--no-input`
generan proyectos válidos con y sin features opcionales, y la matriz completa
pasa `make test-template`.

## Notas

- Orden recomendado: Fase 0 (0.1 → 0.2 → 0.4.1–0.4.7 → 0.3 → 0.4.8–0.4.10) →
  1 → 2 → 3 → 4 → 5. El adelanto de 0.4.1–0.4.7 satisface la dependencia
  `0.3.4 → 0.4.6` (build tag `integration`).
- La Fase 0 y la 1 se pueden entregar juntas como primera iteración.
- Cada fase debe terminar con `make test-template`; es la única forma de evitar
  regresiones como las actuales.
- Todas las pruebas que generen la plantilla deben usar `uvx`, nunca una
  instalación global de cookiecutter.

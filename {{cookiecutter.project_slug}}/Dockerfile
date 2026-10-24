FROM --platform=$BUILDPLATFORM golang:1.27-alpine@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
        -trimpath -buildvcs=false -ldflags="-s -w" -o /build/http ./cmd/http
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
        -trimpath -buildvcs=false -ldflags="-s -w" -o /build/worker ./cmd/worker
{% endif %}
{% if cookiecutter.use_db == "yes" %}
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
        -trimpath -buildvcs=false -ldflags="-s -w" -o /build/devcli ./cmd/devcli
{% endif %}

FROM alpine:3.22.7@sha256:2e0b81d60597f449baa56dc25d355367869d2dca4cd57e866f1e2b52af3eb55f

ARG TARGETARCH
ARG ATLAS_VERSION=1.3.5
ARG ATLAS_SHA256_AMD64=41d2a3c77fab370aead88b9a9faf2bab24d22e3e5728dc2dd7f87404adbce048
ARG ATLAS_SHA256_ARM64=bcdb097755995ef304a08635d38b65e9c2636489f45d0f7c04f2316bbd888ba9

WORKDIR /app

RUN apk add --no-cache ca-certificates tini \
        && addgroup -S app \
        && adduser -S -G app app

{% if cookiecutter.use_db == "yes" %}
RUN set -eu; \
        case "${TARGETARCH}" in \
            amd64) atlas_sha="${ATLAS_SHA256_AMD64}" ;; \
            arm64) atlas_sha="${ATLAS_SHA256_ARM64}" ;; \
            *) echo "arquitectura no soportada: ${TARGETARCH}" >&2; exit 1 ;; \
        esac; \
        wget -qO /tmp/atlas "https://release.ariga.io/atlas/atlas-linux-${TARGETARCH}-v${ATLAS_VERSION}"; \
        echo "${atlas_sha}  /tmp/atlas" | sha256sum -c -; \
        install -m 0755 /tmp/atlas /usr/local/bin/atlas; \
        rm -f /tmp/atlas
{% endif %}

COPY --from=builder --chown=app:app /build/ /app/
COPY --chown=app:app env/ env/
{% if cookiecutter.use_db == "yes" %}
COPY --chown=app:app migrations/ migrations/
{% endif %}

COPY --chown=app:app entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh

USER app
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
        CMD wget -qO- "http://127.0.0.1:${PORT:-8080}/healthz" > /dev/null || exit 1

ENTRYPOINT ["/sbin/tini", "--", "./entrypoint.sh"]

FROM golang:1.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /build/http ./cmd/http
{% if cookiecutter.use_worker == "yes" and cookiecutter.use_asynq == "yes" %}
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /build/worker ./cmd/worker
{% endif %}
{% if cookiecutter.use_db == "yes" %}
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /build/devcli ./cmd/devcli
{% endif %}

FROM alpine:latest
WORKDIR /app

{% if cookiecutter.use_db == "yes" %}
RUN wget https://release.ariga.io/atlas/atlas-linux-amd64-latest -O /usr/local/bin/atlas \
        && chmod +x /usr/local/bin/atlas
{% endif %}

COPY --from=builder /build/ /app/
COPY env/ env/
{% if cookiecutter.use_db == "yes" %}
COPY migrations/ migrations/
{% endif %}

COPY entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh
EXPOSE 8080

CMD ["./entrypoint.sh"]

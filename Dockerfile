FROM golang:1.24-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/trackmate-api ./cmd/trackmate-api \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/trackmate-worker ./cmd/trackmate-worker \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/trackmate-healthcheck ./cmd/trackmate-healthcheck

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /out/migrate /usr/local/bin/migrate
COPY --from=build /out/trackmate-api /usr/local/bin/trackmate-api
COPY --from=build /out/trackmate-worker /usr/local/bin/trackmate-worker
COPY --from=build /out/trackmate-healthcheck /usr/local/bin/trackmate-healthcheck
COPY migrations ./migrations

CMD ["trackmate-api"]

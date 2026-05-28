# Trackmate

Trackmate is a Telegram accountability bot for one daily focus task per participant and a shared progress stream.

Trackmate 2.0 is Go-only. The active local runtime is:

- `trackmate-api`: Telegram long-polling API process.
- `trackmate-worker`: daily task transitions, alert dispatch, progress outbox publishing, and local-only E2E control.
- `migrate`: goose migrations.
- PostgreSQL: primary state store.

The product owns two forum topics:

- `Сегодня`
- `Прогресс`

The old Materials feature was removed from runtime and schema. Migrations preserve daily tasks, reports, alerts, participants, workspaces, topic bindings for Today/Progress, and non-material progress events. Material batches/items/progress and the Materials topic binding are intentionally dropped.

## Environment

Copy `.env.example` to `.env` and set `TRACKMATE__BOT_TOKEN`.

For local Docker, `docker-compose.yml` overrides `TRACKMATE__DATABASE_URL` to the Compose PostgreSQL service. PostgreSQL is published on `127.0.0.1:5432` by default.

Important variables:

- `TRACKMATE__BOT_TOKEN`
- `TRACKMATE__DATABASE_URL`
- `TRACKMATE__DEFAULT_TIMEZONE`
- `TRACKMATE__WORKER_TICK_SECONDS`
- `TRACKMATE__ENVIRONMENT`
- `TRACKMATE__CONTROL_HTTP_ADDR` for non-production E2E control

Control endpoints are disabled when `TRACKMATE__ENVIRONMENT=production`.

## Quick Start

```bash
make setup
cp .env.example .env
make docker-up
docker compose ps
```

For local non-Docker processes:

```bash
make migrate
make api
```

In a second shell:

```bash
make worker
```

## Commands

```bash
make setup               # go mod tidy
make test                # go test ./...
make lint                # gofmt drift check
make migrate             # apply goose migrations
make api                 # run Telegram poller
make worker              # run background worker
make docker-up           # build and start Docker services
make docker-reset        # remove Docker volume and restart stack
make docker-db-backup
make docker-db-backup-stop
make docker-db-restore FILE=backups/trackmate.dump
```

Storage integration tests require a disposable PostgreSQL URL:

```bash
TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./...
```

The tests create and drop a temporary schema under that database.

## Local E2E Control

When the worker runs with `TRACKMATE__CONTROL_HTTP_ADDR`, local-only endpoints are available:

```bash
curl -X POST 'http://127.0.0.1:8082/control/reset?chat_id=-1001234567890'
curl 'http://127.0.0.1:8082/control/topics?chat_id=-1001234567890'
curl -X POST 'http://127.0.0.1:8082/control/clock' \
  -H 'content-type: application/json' \
  -d '{"now":"2026-05-28T12:01:00Z"}'
curl -X POST 'http://127.0.0.1:8082/control/tick'
```

Use these only with disposable local/test Telegram groups and local databases.

## Repository Layout

- `cmd/`: process entrypoints.
- `internal/telegram/`: typed Bot API client, update structs, retry/error semantics, input extraction.
- `internal/dispatcher/`: per-mailbox update ordering.
- `internal/bot/`: update routing and Telegram-facing handlers.
- `internal/app/`: setup, Today transition, and progress publishing use cases.
- `internal/storage/postgres/`: pgx storage, transactions, claims, control helpers.
- `internal/ui/`: Telegram HTML formatters and inline keyboards.
- `migrations/`: goose migrations.
- `e2e/telegram/`: Trackmate-owned JSONL scenarios for the sibling E2E runner.
- `docs/`: architecture and cutover notes.

## Documentation

- [docs/current-architecture.md](docs/current-architecture.md)
- [docs/go-migration.md](docs/go-migration.md)
- [docs/go-cutover-runbook.md](docs/go-cutover-runbook.md)
- [docs/adr/0001-remove-materials-topic.md](docs/adr/0001-remove-materials-topic.md)

Keep machine-specific operational details in `private-docs/`, not in tracked docs.

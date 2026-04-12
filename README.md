# Trackmate

Trackmate is a Telegram accountability bot for shared materials, one daily focus task per participant, and a progress stream that keeps the group honest.

## At a glance

- Telegram bot with a separate background worker
- Admin custom progress updates via `/update` in the `Прогресс` topic
- PostgreSQL-backed state and Alembic migrations
- local `uv` workflow for development
- Docker workflow for long-running deployment
- explicit backup, restore, reset, and update commands

## Runtime model

Trackmate runs as three main components:

- `trackmate-api`
  Aiogram polling process that handles Telegram updates.

- `trackmate-worker`
  Periodic background jobs for alerts and scheduled processing.

- `PostgreSQL`
  Primary state store.

## Environment model

- Local `.env` plus local `uv` or local Docker runs should be treated as the development environment by default.
- A VPS deployment with its own `.env` and long-running polling worker should be treated as the production environment.
- PostgreSQL is published on `127.0.0.1:5432` by default, so it is reachable from the same machine but not exposed on the host network.

## Repository layout

- `src/trackmate/`
  Application code.

- `src/trackmate/adapters/`
  Persistence and Telegram integration layers.

- `src/trackmate/application/`
  Use-case orchestration and application services.

- `src/trackmate/domain/`
  Domain rules and business logic.

- `src/trackmate/entrypoints/`
  API and worker startup modules.

- `tests/`
  Automated test suite.

- `scripts/`
  Database reset, backup, restore, and Docker update helpers.

- `docs/`
  Public tracked documentation.

- `private-docs/`
  Local-only operational notes for the current machine or deployment.

## Requirements

- Python `3.14`
- `uv`
- PostgreSQL for local non-Docker development
- Docker and Docker Compose for containerized runs

## Quick start

### Local development

```bash
make setup
cp .env.example .env
uv run alembic upgrade head
make api
```

In a second shell:

```bash
make worker
```

### Docker development

```bash
cp .env.example .env
make docker-up
```

For Docker you usually only need to set `TRACKMATE__BOT_TOKEN` in `.env`. `docker-compose.yml` overrides the database URL for containers.

The `migrate` service runs `alembic upgrade head` before `api` and `worker` start.

## Development commands

```bash
make setup               # install dependencies
make api                 # run local Telegram polling process
make worker              # run background worker
make lint                # run ruff
make test                # run pytest
make db-reset            # reset local database from .env connection
make docker-reset        # reset Docker database and restart the stack
make docker-up           # build and start Docker services
make docker-update       # pull, rebuild, restart, and wait for health
make docker-db-backup    # create Docker Postgres backup
make docker-db-backup-stop
make docker-db-restore FILE=backups/trackmate.dump
```

## Deployment flow

On a production machine that tracks an upstream branch:

```bash
make docker-update
```

It will:

- run `git pull --ff-only` if the branch has an upstream;
- rebuild and restart the Docker services;
- wait until `postgres`, `api`, and `worker` are ready.

Recommended production update flow:

1. Validate the change locally.
2. Commit and push the change.
3. On the production machine, update the checked-out branch and restart with `make docker-update`.
4. Verify `docker compose ps` and follow logs after restart.

If you are doing a machine move or database cutover, use [docs/migration.md](docs/migration.md) instead of the standard update flow.

## Database operations

For a clean local database reset against the current `TRACKMATE__DATABASE_URL` from `.env`:

```bash
make db-reset
```

For a full Docker reset with PostgreSQL volume removal, migrations, and service restart:

```bash
make docker-reset
```

For a Docker database backup:

```bash
make docker-db-backup
```

For a final backup that stops the app before cutover:

```bash
make docker-db-backup-stop
```

For a Docker database restore:

```bash
make docker-db-restore FILE=backups/trackmate.dump
```

## Documentation

- [docs/README.md](docs/README.md)
  Public documentation index.

- [docs/migration.md](docs/migration.md)
  Migration and machine cutover runbook.

## Notes

- Keep machine-specific operational details in `private-docs/`, not in tracked public docs.
- Do not broaden the PostgreSQL host bind unless you explicitly need remote database access.

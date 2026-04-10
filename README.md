# Trackmate

Trackmate is a Telegram accountability bot for shared materials, one daily focus task per participant, and a progress stream that keeps the group honest.

## Runtime

- `trackmate-api`: aiogram polling process
- `trackmate-worker`: periodic background jobs
- `PostgreSQL`: primary state store

## Quick start

Local run against a locally exposed PostgreSQL:

```bash
uv sync
cp .env.example .env
uv run alembic upgrade head
uv run python -m trackmate.entrypoints.api
```

In a second shell:

```bash
uv run python -m trackmate.entrypoints.worker
```

## Docker run

```bash
cp .env.example .env
docker compose up -d --build
```

For Docker you usually only need to set `TRACKMATE__BOT_TOKEN` in `.env`. `docker-compose.yml` overrides the database URL for containers.

`migrate` runs `alembic upgrade head` before `api` and `worker` start.

## Environment model

- Local `.env` plus local `uv` or local Docker runs should be treated as the development environment by default.
- A VPS deployment with its own `.env` and long-running polling worker should be treated as the production environment.
- PostgreSQL is published on `127.0.0.1:5432` by default, so it is reachable from the same machine but not exposed on the host network.

## One-command update

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

## Development helpers

For a clean local database reset against the current `TRACKMATE__DATABASE_URL` from `.env`:

```bash
make db-reset
```

For a full Docker reset with PostgreSQL volume removal, migrations, and service restart:

```bash
make docker-reset
```

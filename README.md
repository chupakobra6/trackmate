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

## One-command update

On the machine where the bot is already running:

```bash
make docker-update
```

It will:
- run `git pull --ff-only` if the branch has an upstream;
- rebuild and restart the Docker services;
- wait until `postgres`, `api`, and `worker` are ready.

## Development helpers

For a clean local database reset against the current `TRACKMATE__DATABASE_URL` from `.env`:

```bash
make db-reset
```

For a full Docker reset with PostgreSQL volume removal, migrations, and service restart:

```bash
make docker-reset
```

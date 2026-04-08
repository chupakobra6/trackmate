#!/usr/bin/env sh
set -eu

echo "Resetting docker postgres volume..."

docker compose down -v
docker compose up -d postgres
docker compose build api worker
docker compose run --rm api uv run alembic upgrade head
docker compose up -d api worker
docker compose ps

echo "Docker database reset complete."

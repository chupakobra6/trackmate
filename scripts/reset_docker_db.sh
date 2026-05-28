#!/usr/bin/env sh
set -eu

echo "Resetting docker postgres volume..."

docker compose down -v
docker compose up -d --build
docker compose ps

echo "Docker database reset complete."

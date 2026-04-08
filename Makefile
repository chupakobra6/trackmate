setup:
	uv sync

db-reset:
	sh scripts/reset_dev_db.sh

docker-reset:
	sh scripts/reset_docker_db.sh

lint:
	uv run ruff check .

test:
	uv run pytest

api:
	uv run python -m trackmate.entrypoints.api

worker:
	uv run python -m trackmate.entrypoints.worker

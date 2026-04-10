setup:
	uv sync

db-reset:
	sh scripts/reset_dev_db.sh

docker-reset:
	sh scripts/reset_docker_db.sh

docker-up:
	docker compose up -d --build

docker-update:
	sh scripts/update_docker_app.sh

docker-db-backup:
	sh scripts/backup_docker_db.sh

docker-db-backup-stop:
	sh scripts/backup_docker_db.sh --stop-app

docker-db-restore:
	@test -n "$(FILE)" || (echo "FILE is required. Example: make docker-db-restore FILE=backups/trackmate.dump" && exit 1)
	sh scripts/restore_docker_db.sh "$(FILE)"

lint:
	uv run ruff check .

test:
	uv run pytest

api:
	uv run python -m trackmate.entrypoints.api

worker:
	uv run python -m trackmate.entrypoints.worker

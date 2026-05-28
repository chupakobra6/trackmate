.DEFAULT_GOAL := help

.PHONY: help setup tidy fmt fmt-check lint test migrate api worker dev docker-up docker-reset docker-db-backup docker-db-backup-stop docker-db-restore clean-legacy down logs logs-all logs-db

help:
	@printf "Available commands:\n"
	@printf "  make setup              # go mod tidy\n"
	@printf "  make test               # go test ./...\n"
	@printf "  make lint               # fail on gofmt drift under cmd/ and internal/\n"
	@printf "  make migrate            # apply Go goose migrations\n"
	@printf "  make api                # run Go Telegram poller locally\n"
	@printf "  make worker             # run Go worker locally\n"
	@printf "  make docker-up          # build and start local Docker stack\n"
	@printf "  make clean-legacy       # remove old Python caches from working tree\n"

setup: tidy

tidy:
	go mod tidy

fmt:
	gofmt -w ./cmd ./internal

fmt-check:
	@out="$$(gofmt -l ./cmd ./internal)"; \
	if [ -n "$$out" ]; then \
		echo "gofmt drift detected:"; \
		echo "$$out"; \
		exit 1; \
	fi

lint: fmt-check

test:
	go test ./...

migrate:
	go run ./cmd/migrate

api:
	go run ./cmd/trackmate-api

worker:
	go run ./cmd/trackmate-worker

dev: docker-up

docker-up:
	docker compose up -d --build

docker-reset:
	sh scripts/reset_docker_db.sh

docker-db-backup:
	sh scripts/backup_docker_db.sh

docker-db-backup-stop:
	sh scripts/backup_docker_db.sh --stop-app

docker-db-restore:
	@test -n "$(FILE)" || (echo "FILE is required. Example: make docker-db-restore FILE=backups/trackmate.dump" && exit 1)
	sh scripts/restore_docker_db.sh "$(FILE)"

clean-legacy:
	rm -rf .venv .pytest_cache .ruff_cache .mypy_cache .coverage htmlcov
	find . -type d -name '__pycache__' -prune -exec rm -rf {} +

down:
	docker compose down

logs:
	docker compose logs --tail=200 -f api worker migrate

logs-all:
	docker compose logs --tail=200 -f

logs-db:
	docker compose logs --tail=200 -f postgres

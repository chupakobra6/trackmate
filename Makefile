.DEFAULT_GOAL := help

LOCAL_COMPOSE_FILES := docker-compose.yml:docker-compose.local.yml
LOCAL_COMPOSE := COMPOSE_FILE=$(LOCAL_COMPOSE_FILES) docker compose

.PHONY: help setup tidy fmt fmt-check lint vet test check migrate api worker dev docker-up docker-cache-prune docker-reset docker-db-backup docker-db-backup-stop docker-db-restore clean down logs logs-all logs-db

help:
	@printf "Available commands:\n"
	@printf "  make setup              # go mod tidy\n"
	@printf "  make test               # go test ./...\n"
	@printf "  make lint               # fail on gofmt drift under cmd/ and internal/\n"
	@printf "  make vet                # go vet ./...\n"
	@printf "  make check              # lint, vet, and test\n"
	@printf "  make migrate            # apply Go goose migrations\n"
	@printf "  make api                # run Go Telegram poller locally\n"
	@printf "  make worker             # run Go worker locally\n"
	@printf "  make docker-up          # build and start local Docker stack\n"
	@printf "  make docker-cache-prune # cap unused local BuildKit cache at 2 GB\n"
	@printf "  make docker-reset       # remove local Docker DB volume and restart stack\n"
	@printf "  make docker-db-backup   # create logical PostgreSQL dump\n"
	@printf "  make docker-db-backup-stop  # backup after stopping api and worker\n"
	@printf "  make docker-db-restore FILE=backups/trackmate.dump\n"
	@printf "  make down               # stop local Docker stack\n"
	@printf "  make logs               # follow api, worker, and migrate logs\n"
	@printf "  make logs-all           # follow all Docker Compose logs\n"
	@printf "  make logs-db            # follow PostgreSQL logs\n"
	@printf "  make clean              # remove generated local artifacts\n"

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

vet:
	go vet ./...

test:
	go test ./...

check: lint vet test

migrate:
	go run ./cmd/migrate

api:
	go run ./cmd/trackmate-api

worker:
	go run ./cmd/trackmate-worker

dev: docker-up

docker-up:
	$(LOCAL_COMPOSE) up -d --build
	$(MAKE) docker-cache-prune

docker-cache-prune:
	docker buildx prune --force --max-used-space 2GB

docker-reset:
	COMPOSE_FILE=$(LOCAL_COMPOSE_FILES) sh scripts/reset_docker_db.sh

docker-db-backup:
	sh scripts/backup_docker_db.sh

docker-db-backup-stop:
	sh scripts/backup_docker_db.sh --stop-app

docker-db-restore:
	@test -n "$(FILE)" || (echo "FILE is required. Example: make docker-db-restore FILE=backups/trackmate.dump" && exit 1)
	sh scripts/restore_docker_db.sh "$(FILE)"

clean:
	rm -rf tmp

down:
	$(LOCAL_COMPOSE) down

logs:
	$(LOCAL_COMPOSE) logs --tail=200 -f api worker migrate

logs-all:
	$(LOCAL_COMPOSE) logs --tail=200 -f

logs-db:
	$(LOCAL_COMPOSE) logs --tail=200 -f postgres

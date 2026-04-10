#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
cd "$REPO_ROOT"

POSTGRES_SERVICE=${TRACKMATE_POSTGRES_SERVICE:-postgres}
API_SERVICE=${TRACKMATE_API_SERVICE:-api}
WORKER_SERVICE=${TRACKMATE_WORKER_SERVICE:-worker}
MIGRATE_SERVICE=${TRACKMATE_MIGRATE_SERVICE:-migrate}
DB_NAME=${TRACKMATE_DB_NAME:-trackmate}
DB_USER=${TRACKMATE_DB_USER:-postgres}

TEMP_ARCHIVE=""

usage() {
  cat <<'EOF'
Usage: sh scripts/restore_docker_db.sh DUMP_FILE

Restores a logical Postgres dump into the Docker Compose postgres service.
The target database is dropped and recreated before restore.
EOF
}

validate_service_name() {
  case "$2" in
    *[!A-Za-z0-9_-]* | "")
      echo "Invalid $1: $2" >&2
      exit 1
      ;;
  esac
}

validate_identifier() {
  case "$2" in
    *[!A-Za-z0-9_]* | "")
      echo "Invalid $1: $2" >&2
      exit 1
      ;;
  esac
}

is_absolute_path() {
  case "$1" in
    /* | [A-Za-z]:/*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

resolve_path() {
  if is_absolute_path "$1"; then
    printf '%s\n' "$1"
  else
    printf '%s/%s\n' "$REPO_ROOT" "$1"
  fi
}

container_id() {
  docker compose ps -q "$1" 2>/dev/null || true
}

wait_for_service() {
  service_name=$1
  attempts=${2:-60}
  while [ "$attempts" -gt 0 ]; do
    cid=$(container_id "$service_name")
    if [ -n "$cid" ]; then
      status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$cid" 2>/dev/null || true)
      if [ "$status" = "healthy" ] || [ "$status" = "running" ]; then
        return 0
      fi
    fi
    attempts=$((attempts - 1))
    sleep 2
  done
  echo "Timed out waiting for service '$service_name' to become ready." >&2
  exit 1
}

cleanup() {
  if [ -n "$TEMP_ARCHIVE" ]; then
    cid=$(container_id "$POSTGRES_SERVICE")
    if [ -n "$cid" ]; then
      MSYS_NO_PATHCONV=1 docker exec "$cid" rm -f "$TEMP_ARCHIVE" >/dev/null 2>&1 || true
    fi
  fi
}

trap cleanup EXIT INT TERM

validate_service_name "postgres service" "$POSTGRES_SERVICE"
validate_service_name "api service" "$API_SERVICE"
validate_service_name "worker service" "$WORKER_SERVICE"
validate_service_name "migrate service" "$MIGRATE_SERVICE"
validate_identifier "database name" "$DB_NAME"
validate_identifier "database user" "$DB_USER"

[ $# -eq 1 ] || {
  usage >&2
  exit 1
}

DUMP_PATH=$(resolve_path "$1")
[ -f "$DUMP_PATH" ] || {
  echo "Dump file not found: $DUMP_PATH" >&2
  exit 1
}

echo "Verifying dump archive: $DUMP_PATH"
docker compose up -d "$POSTGRES_SERVICE" >/dev/null
wait_for_service "$POSTGRES_SERVICE" 90

docker compose stop "$API_SERVICE" "$WORKER_SERVICE" >/dev/null || true

POSTGRES_CID=$(container_id "$POSTGRES_SERVICE")
[ -n "$POSTGRES_CID" ] || {
  echo "Unable to locate container for service '$POSTGRES_SERVICE'." >&2
  exit 1
}

TEMP_ARCHIVE="/tmp/trackmate_restore_$$.dump"
docker cp "$DUMP_PATH" "$POSTGRES_CID:$TEMP_ARCHIVE"
MSYS_NO_PATHCONV=1 docker compose exec -T "$POSTGRES_SERVICE" pg_restore --list "$TEMP_ARCHIVE" >/dev/null

echo "Resetting database '$DB_NAME'"
docker compose exec -T "$POSTGRES_SERVICE" psql -U "$DB_USER" -d postgres -v ON_ERROR_STOP=1 \
  -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid();"
docker compose exec -T "$POSTGRES_SERVICE" psql -U "$DB_USER" -d postgres -v ON_ERROR_STOP=1 \
  -c "DROP DATABASE IF EXISTS \"$DB_NAME\";"
docker compose exec -T "$POSTGRES_SERVICE" psql -U "$DB_USER" -d postgres -v ON_ERROR_STOP=1 \
  -c "CREATE DATABASE \"$DB_NAME\";"

echo "Restoring dump into '$DB_NAME'"
MSYS_NO_PATHCONV=1 docker compose exec -T "$POSTGRES_SERVICE" pg_restore \
  -U "$DB_USER" \
  -d "$DB_NAME" \
  --clean \
  --if-exists \
  --no-owner \
  --no-privileges \
  --exit-on-error \
  "$TEMP_ARCHIVE"

echo "Running migrations"
docker compose run --rm "$MIGRATE_SERVICE" >/dev/null

echo "Starting application services"
docker compose up -d "$API_SERVICE" "$WORKER_SERVICE" >/dev/null
wait_for_service "$API_SERVICE" 90
wait_for_service "$WORKER_SERVICE" 90

echo "Restore complete."

#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
cd "$REPO_ROOT"

POSTGRES_SERVICE=${TRACKMATE_POSTGRES_SERVICE:-postgres}
API_SERVICE=${TRACKMATE_API_SERVICE:-api}
WORKER_SERVICE=${TRACKMATE_WORKER_SERVICE:-worker}
DB_NAME=${TRACKMATE_DB_NAME:-trackmate}
DB_USER=${TRACKMATE_DB_USER:-postgres}

OUTPUT_PATH=""
FORCE=0
STOP_APP=0
SUCCESS=0
API_WAS_RUNNING=0
WORKER_WAS_RUNNING=0
TEMP_ARCHIVE=""

usage() {
  cat <<'EOF'
Usage: sh scripts/backup_docker_db.sh [--output PATH] [--force] [--stop-app]

Creates a logical Postgres dump from the Docker Compose postgres service.

Options:
  --output PATH  Write the dump to PATH. Defaults to backups/trackmate_<UTC timestamp>.dump
  --force        Allow overwriting an existing output path.
  --stop-app     Stop api/worker before dumping. On success they stay stopped for cutover.
  --help         Show this help.
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

service_exists() {
  docker compose config --services | grep -qx "$1"
}

container_id() {
  docker compose ps -q "$1" 2>/dev/null || true
}

is_service_running() {
  cid=$(container_id "$1")
  [ -n "$cid" ] || return 1
  state=$(docker inspect --format '{{.State.Status}}' "$cid" 2>/dev/null || true)
  [ "$state" = "running" ]
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

write_checksum() {
  target=$1
  target_dir=$(dirname "$target")
  target_name=$(basename "$target")
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$target_dir" && sha256sum "$target_name") > "$target.sha256"
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$target_dir" && shasum -a 256 "$target_name") > "$target.sha256"
  elif command -v openssl >/dev/null 2>&1; then
    checksum=$(openssl dgst -sha256 -r "$target" | awk '{print $1}')
    printf '%s  %s\n' "$checksum" "$target_name" > "$target.sha256"
  fi
}

cleanup() {
  if [ -n "$TEMP_ARCHIVE" ]; then
    cid=$(container_id "$POSTGRES_SERVICE")
    if [ -n "$cid" ]; then
      MSYS_NO_PATHCONV=1 docker exec "$cid" rm -f "$TEMP_ARCHIVE" >/dev/null 2>&1 || true
    fi
  fi

  if [ "$STOP_APP" -ne 1 ] || [ "$SUCCESS" -eq 1 ]; then
    return
  fi

  services=""
  if [ "$API_WAS_RUNNING" -eq 1 ]; then
    services="$services $API_SERVICE"
  fi
  if [ "$WORKER_WAS_RUNNING" -eq 1 ]; then
    services="$services $WORKER_SERVICE"
  fi
  if [ -n "$services" ]; then
    echo "Backup failed. Restarting previously running services:$services" >&2
    docker compose up -d $services >/dev/null
  fi
}

trap cleanup EXIT INT TERM

validate_service_name "postgres service" "$POSTGRES_SERVICE"
validate_service_name "api service" "$API_SERVICE"
validate_service_name "worker service" "$WORKER_SERVICE"
validate_identifier "database name" "$DB_NAME"
validate_identifier "database user" "$DB_USER"

while [ $# -gt 0 ]; do
  case "$1" in
    --output)
      [ $# -ge 2 ] || {
        echo "--output requires a value." >&2
        exit 1
      }
      OUTPUT_PATH=$2
      shift 2
      ;;
    --force)
      FORCE=1
      shift
      ;;
    --stop-app)
      STOP_APP=1
      shift
      ;;
    --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

timestamp=$(date -u +%Y%m%dT%H%M%SZ)
if [ -z "$OUTPUT_PATH" ]; then
  OUTPUT_PATH="$REPO_ROOT/backups/trackmate_${timestamp}.dump"
else
  OUTPUT_PATH=$(resolve_path "$OUTPUT_PATH")
fi

OUTPUT_DIR=$(dirname "$OUTPUT_PATH")
OUTPUT_NAME=$(basename "$OUTPUT_PATH")
mkdir -p "$OUTPUT_DIR"

if [ -e "$OUTPUT_PATH" ] && [ "$FORCE" -ne 1 ]; then
  echo "Refusing to overwrite existing file: $OUTPUT_PATH" >&2
  exit 1
fi

docker compose up -d "$POSTGRES_SERVICE" >/dev/null
wait_for_service "$POSTGRES_SERVICE" 90

if [ "$STOP_APP" -eq 1 ]; then
  if service_exists "$API_SERVICE" && is_service_running "$API_SERVICE"; then
    API_WAS_RUNNING=1
  fi
  if service_exists "$WORKER_SERVICE" && is_service_running "$WORKER_SERVICE"; then
    WORKER_WAS_RUNNING=1
  fi

  services_to_stop=""
  if [ "$API_WAS_RUNNING" -eq 1 ]; then
    services_to_stop="$services_to_stop $API_SERVICE"
  fi
  if [ "$WORKER_WAS_RUNNING" -eq 1 ]; then
    services_to_stop="$services_to_stop $WORKER_SERVICE"
  fi
  if [ -n "$services_to_stop" ]; then
    echo "Stopping application services for cutover backup:$services_to_stop"
    docker compose stop $services_to_stop >/dev/null
  fi
fi

echo "Creating dump: $OUTPUT_PATH"
docker compose exec -T "$POSTGRES_SERVICE" pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc > "$OUTPUT_PATH"

POSTGRES_CID=$(container_id "$POSTGRES_SERVICE")
[ -n "$POSTGRES_CID" ] || {
  echo "Unable to locate container for service '$POSTGRES_SERVICE'." >&2
  exit 1
}

TEMP_ARCHIVE="/tmp/trackmate_backup_verify_$$.dump"
docker cp "$OUTPUT_PATH" "$POSTGRES_CID:$TEMP_ARCHIVE"
MSYS_NO_PATHCONV=1 docker compose exec -T "$POSTGRES_SERVICE" pg_restore --list "$TEMP_ARCHIVE" >/dev/null

cat > "$OUTPUT_PATH.meta" <<EOF
created_at_utc=$timestamp
git_commit=$(git rev-parse HEAD 2>/dev/null || echo unknown)
postgres_service=$POSTGRES_SERVICE
database_name=$DB_NAME
database_user=$DB_USER
postgres_version=$(docker compose exec -T "$POSTGRES_SERVICE" postgres --version | tr -d '\r')
cutover_backup=$STOP_APP
dump_file=$OUTPUT_NAME
EOF

write_checksum "$OUTPUT_PATH"

SUCCESS=1

echo "Backup complete."
echo "Dump: $OUTPUT_PATH"
echo "Metadata: $OUTPUT_PATH.meta"
if [ -f "$OUTPUT_PATH.sha256" ]; then
  echo "Checksum: $OUTPUT_PATH.sha256"
fi
if [ "$STOP_APP" -eq 1 ]; then
  echo "Application services remain stopped after this cutover backup."
fi

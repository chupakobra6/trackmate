#!/usr/bin/env sh
set -eu

wait_for_service() {
	service="$1"
	attempts="${2:-60}"
	index=0

	while [ "$index" -lt "$attempts" ]; do
		container_id="$(docker compose ps -q "$service")"
		if [ -n "$container_id" ]; then
			status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container_id" 2>/dev/null || true)"
			if [ "$status" = "healthy" ] || [ "$status" = "running" ]; then
				echo "$service: $status"
				return 0
			fi
		fi
		index=$((index + 1))
		sleep 2
	done

	echo "Timed out waiting for $service." >&2
	docker compose ps >&2
	exit 1
}

echo "Updating repository..."
if git rev-parse --abbrev-ref --symbolic-full-name '@{u}' >/dev/null 2>&1; then
	git pull --ff-only
else
	echo "No upstream branch configured, skipping git pull."
fi

echo "Rebuilding and starting containers..."
docker compose up -d --build

echo "Waiting for services..."
wait_for_service postgres
wait_for_service api
wait_for_service worker

docker compose ps

echo "Docker app update complete."

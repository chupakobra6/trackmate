# Operations

This document covers current operational workflows for the Go runtime: local
validation, logical PostgreSQL backup/restore, production update safety, and
Telegram E2E.

Do not run these commands against production by accident. Check the current
`.env`, `docker context`, and git commit before touching a long-lived bot.

## Local Docker

Fresh local start:

```bash
cp .env.example .env
make docker-up
docker compose ps
docker compose logs --tail=200 migrate api worker
```

Full local database reset:

```bash
make docker-reset
```

`docker-reset` removes the local PostgreSQL Docker volume. Do not use it on a
machine where the Docker volume contains data you need.

## Logical Backups

Use logical PostgreSQL dumps, not Docker volume copies.

Routine backup while the bot is running:

```bash
make docker-db-backup
```

Final backup before moving or replacing a running bot:

```bash
make docker-db-backup-stop
```

`docker-db-backup-stop` stops `api` and `worker`, writes the dump, verifies it,
and leaves application services stopped after success. This prevents the old
polling process from writing new data after the final dump.

Backups are written under `backups/` by default. That directory is ignored by
git.

## Restore

Restore a dump into the Compose PostgreSQL service:

```bash
make docker-db-restore FILE=backups/trackmate_20260528T120000Z.dump
```

The restore script:

- verifies the dump archive with `pg_restore --list`;
- starts Docker PostgreSQL;
- stops `api` and `worker`;
- drops and recreates the target database;
- restores the dump;
- runs `migrate`;
- starts `api` and `worker`;
- waits for both services to become healthy.

## Production Update Shape

For a data-preserving update on the machine that already runs the bot:

1. Confirm the current commit and Docker state.

```bash
git rev-parse --short HEAD
docker compose ps
```

2. Create a backup. Use the stop-app variant when a write freeze is required.

```bash
make docker-db-backup-stop
```

3. Pull the intended code and rebuild.

```bash
git pull --ff-only
docker compose up -d --build
```

4. Verify health and logs.

```bash
docker compose ps
docker compose logs --tail=200 migrate api worker
```

5. If the update is verified, keep the old backup until the next stable backup
   cycle.

Never run two Telegram polling runtimes for the same bot token at the same time.

## Schema Checks

Useful read-only checks after restore or update:

```sql
SELECT count(*) FROM workspace_groups;
SELECT count(*) FROM participants;
SELECT count(*) FROM daily_tasks;
SELECT count(*) FROM daily_task_alerts;
SELECT count(*) FROM progress_events;
SELECT count(*) FROM routine_plans;
SELECT count(*) FROM routine_checkins;
SELECT count(*) FROM seasonal_goal_sets;
SELECT count(*) FROM pending_inputs;
```

Materials must not exist in the active schema:

```sql
SELECT to_regclass('public.material_batches');
SELECT to_regclass('public.material_items');
SELECT to_regclass('public.material_participant_progresses');
SELECT enumlabel
FROM pg_enum e
JOIN pg_type t ON t.oid = e.enumtypid
WHERE t.typname = 'topickey';
SELECT enumlabel
FROM pg_enum e
JOIN pg_type t ON t.oid = e.enumtypid
WHERE t.typname = 'progresseventtype';
```

Expected:

- material tables return `NULL`;
- `topickey` contains `today`, `progress`, `routine`, `goals`;
- `progresseventtype` contains `daily_task.closed`, `daily_task.auto_failed`,
  `system_alert`, `custom_update`.

## Manual Production Data Fixes

Manual data fixes are production operations, not local development. Keep them
targeted and auditable:

1. Identify the workspace, topic, row IDs, and Telegram message IDs from the
   production database or logs.
2. Stop `api` and `worker` when the operation changes live state or deletes
   rows that workers could touch.
3. Create a logical backup before any write:

```bash
make docker-db-backup-stop
```

4. Change only the scoped rows needed for the incident. Do not reset unrelated
   Today, Goals, Progress, or participant history while fixing routine data.
5. Delete Telegram messages only by known IDs from storage/logs. If Bot API says
   a message is already missing, record that and continue; do not bulk-delete a
   topic to compensate for missing IDs.
6. Verify with read-only SQL counters, restart `api` and `worker`, then check
   `docker compose ps` and recent logs.

Record the backup path, before/after counts, Telegram delete/edit results, and
service health in the project handoff or incident notes.

## Telegram E2E

Runner repository:

```bash
cd /Users/igor/projects/telegram-bot-e2e-test-tool
go test ./...
make fixtures
```

Trackmate scenarios live in `e2e/telegram/scenarios`. Render the templates as
described in [../e2e/telegram/README.md](../e2e/telegram/README.md), then run
the current suite against a disposable forum group.

After a run, clean visible test messages while keeping topics and pinned
intro/control messages:

```bash
CHAT="$TRACKMATE_CHAT" go run ./cmd/tg-e2e-tool run-scenario \
  /Users/igor/projects/trackmate/tmp/e2e-rendered/99-cleanup-visible-messages.jsonl
```

For deterministic progress and alert checks, use the local-only control API:

```bash
curl -fsS -X POST 'http://127.0.0.1:8082/control/reset?chat_id=<bot-api-chat-id>'
curl -fsS -X POST 'http://127.0.0.1:8082/control/clock' \
  -H 'content-type: application/json' \
  -d '{"now":"2026-05-29T00:05:00Z"}'
curl -fsS -X POST 'http://127.0.0.1:8082/control/tick'
curl -fsS -X POST 'http://127.0.0.1:8082/control/clock' \
  -H 'content-type: application/json' \
  -d '{}'
```

Control endpoints are available only outside production.

# Go Cutover Runbook

This runbook is for local/test validation and for a future controlled database-preserving cutover. Do not run it against production by accident.

## Preconditions

- Current backup exists before any real cutover.
- `TRACKMATE__ENVIRONMENT` is not `production` for local validation.
- Bot token and database URL are supplied through `.env` or deployment secrets.
- The target Telegram group is a test forum supergroup when running E2E.

## Local Fresh-DB Validation

```bash
cp .env.example .env
make docker-reset
docker compose ps
docker compose logs --tail=200 migrate api worker
```

Expected:

- `migrate` logs `migrations_applied`;
- `postgres`, `api`, and `worker` are healthy;
- no Python process is required.

## Existing DB Migration Check

For a database copied from the old runtime:

```bash
make docker-db-backup-stop
make docker-db-restore FILE=backups/<dump>.dump
docker compose run --rm migrate
```

After migration, verify retained product data:

```sql
SELECT count(*) FROM workspace_groups;
SELECT count(*) FROM participants;
SELECT count(*) FROM daily_tasks;
SELECT count(*) FROM daily_task_alerts;
SELECT count(*) FROM progress_events;
```

Verify deleted Materials shape:

```sql
SELECT to_regclass('public.material_batches');
SELECT to_regclass('public.material_items');
SELECT to_regclass('public.material_participant_progresses');
SELECT enumlabel FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid WHERE t.typname = 'topickey';
SELECT enumlabel FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid WHERE t.typname = 'progresseventtype';
```

Expected:

- material tables return `NULL`;
- `topickey` contains only `today`, `progress`;
- `progresseventtype` contains only `daily_task.closed`, `daily_task.auto_failed`, `system_alert`, `custom_update`.

## Telegram E2E

Use `/Users/igor/projects/telegram-bot-e2e-test-tool` against a test group.

Minimum scenario set:

```bash
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/e2e/telegram/scenarios/00-setup-smoke.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/02-today-create-task.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/04-report-done.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/07-wrong-topic-pending-ignored.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/09-progress-topic-event.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/10-alert-ack.jsonl
```

Use the control API to reset state, set the clock, and tick the worker for alert/progress determinism:

```bash
curl -fsS -X POST 'http://127.0.0.1:8082/control/reset?chat_id=<bot-api-chat-id>'
curl -fsS -X POST 'http://127.0.0.1:8082/control/clock' -H 'content-type: application/json' -d '{"now":"2026-05-29T00:05:00Z"}'
curl -fsS -X POST 'http://127.0.0.1:8082/control/tick'
curl -fsS -X POST 'http://127.0.0.1:8082/control/clock' -H 'content-type: application/json' -d '{}'
```

## Rollback

Rollback is restore-based:

1. stop API and worker;
2. restore the pre-cutover dump;
3. start the previous runtime only if that is intentionally chosen.

The Materials deletion migration is intentionally irreversible.

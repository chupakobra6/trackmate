# Go Migration Notes

Trackmate 2.0 replaces the old Python/aiogram runtime with a Go-only runtime.

The migration goal is data preservation for the active product, not binary compatibility with old code. The retained data is:

- workspaces;
- Today and Progress topic bindings;
- participants;
- daily tasks and reports;
- daily task alerts;
- pending Today inputs that still make sense at cutover time;
- non-material progress events.

Materials data is intentionally deleted. The feature is no longer part of the product, and carrying its tables, enum values, callback parser, worker behavior, and formatter branches adds complexity without preserving useful data.

## Schema Strategy

Fresh local databases start from `migrations/202605280001_go_baseline.sql`, which creates the Go schema directly.

Existing Python/Alembic databases can be opened by the Go migration command:

1. `202605280001_go_baseline.sql` creates missing Go-owned objects with `IF NOT EXISTS` and keeps existing active-product rows.
2. `202605280002_drop_materials.sql` removes deleted Materials data and tightens enum/table shape:
   - deletes material pending inputs;
   - deletes material progress events;
   - deletes Materials topic bindings;
   - drops `material_batches`, `material_items`, `material_participant_progresses`;
   - drops `progress_events.material_batch_id`;
   - recreates `topickey` without `materials`;
   - recreates `progresseventtype` without material event types;
   - drops material-only enum types.
3. `202605280003_drop_alembic_version.sql` removes the old Alembic metadata table.

The migration does not drop daily-task, participant, alert, or non-material progress history.

## Runtime Mapping

- Python API entrypoint -> `cmd/trackmate-api`
- Python worker entrypoint -> `cmd/trackmate-worker`
- Alembic -> goose via `cmd/migrate`
- aiogram routers -> explicit `internal/bot` handlers plus typed callback parser
- SQLAlchemy repositories -> pgx storage in `internal/storage/postgres`

The Go runtime keeps the important operational properties from the Python version:

- explicit routing layer;
- update context through typed Telegram models;
- per-update transactions for state changes;
- typed callback parsing;
- callback answer guarantee in the API loop;
- Telegram retry/error classification;
- mailbox ordering per workspace/user;
- DB-level idempotency and claims.

## Validation Scope

Required local checks:

```bash
go test ./...
TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./...
docker compose up -d --build
docker compose ps
docker compose logs --tail=200 api worker migrate
```

Required Telegram E2E scenarios:

- setup smoke;
- Today create task;
- report done;
- wrong-topic pending ignored;
- progress topic event;
- alert ack with control clock/tick.

There is no Python test suite after the Go-only cleanup.

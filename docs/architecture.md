# Architecture

Trackmate is a Go Telegram bot backed by PostgreSQL. The runtime is intentionally
small: one long-polling API process, one worker process, and one migration
command.

## Processes

- `cmd/migrate`: applies goose migrations from `migrations/`.
- `cmd/trackmate-api`: polls Telegram Bot API updates and routes them through
  `internal/bot`.
- `cmd/trackmate-worker`: runs periodic ticks for task transitions, alert
  dispatch, progress publishing, and non-production E2E control.
- `cmd/trackmate-healthcheck`: validates local Docker health.

## Runtime Boundaries

- `internal/config`: `.env` loading, defaults, and production guards.
- `internal/logging`: structured `slog` setup.
- `internal/telegram`: typed Bot API client, update models, retry/error
  classification, and input extraction.
- `internal/dispatcher`: mailbox ordering by workspace/user so related updates
  are processed serially.
- `internal/bot`: explicit update router for setup, Today, reports, and alert
  acknowledgements.
- `internal/app/setup`: forum/admin checks and Today/Progress topic repair.
- `internal/app/today`: daily task rules and report state transitions.
- `internal/app/progress`: progress outbox formatting and publishing.
- `internal/storage/postgres`: pgx storage, transactions, idempotency, DB
  claims, advisory worker lock, and E2E control state.
- `internal/ui`: Telegram HTML formatters and inline keyboards.
- `internal/control`: local-only reset/clock/tick/topics HTTP endpoints.

## Product Surface

Trackmate owns exactly two Telegram forum topics:

- `today`: title `Сегодня`; contains the control message, task cards, report
  prompts, and alerts.
- `progress`: title `Прогресс`; contains published progress events.

Setup is idempotent. It creates or repairs only these two topics, stores their
thread IDs and message IDs in PostgreSQL, and does not create a Materials topic.

Materials is deleted from runtime and schema. The bot does not parse
`material:*` callbacks as product actions, does not store material rows, and does
not publish material progress.

## Today Flow

`today:add` creates one pending `daily_task_text` input scoped to the Today
thread. The next message from that user is accepted only if it arrives in that
thread.

Task creation is protected by:

- one pending input per workspace/user;
- one task per participant per local day;
- a block on creating a new task while the previous task is still open.

Task cards stay in Today and include a report button while the task is open.

`task:report:<task_id>` opens the report flow.
`task:status:<task_id>:<done|partial|failed>` stores pending
`daily_task_report`. The next Today message is claimed through the database,
updates the task card, and creates a `daily_task.closed` progress event.

Wrong-topic input is ignored without consuming pending state.

## Worker Flow

Each tick takes a PostgreSQL advisory lock before transitions.

Transitions:

- after local midnight: `active` to `awaiting_report`, plus
  `day_closed_pending_report` alert;
- after local noon: `active` or `awaiting_report` to `failed`, plus
  `overdue_task_failed` alert and `daily_task.auto_failed` progress event.

Alerts and progress events are claimed with `FOR UPDATE SKIP LOCKED`. Telegram
transient failures are requeued; permanent failures are marked failed.

Alert acknowledgement deletes the visible alert card, marks `acknowledged_at`,
and clears the stored Telegram message ID.

## Data Model

Core tables:

- `workspace_groups`
- `topic_bindings`
- `participants`
- `daily_tasks`
- `daily_task_alerts`
- `pending_inputs`
- `progress_events`
- `app_clock`

Important enum values:

- `topickey`: `today`, `progress`
- `dailytaskstatus`: `active`, `awaiting_report`, `done`, `partial`, `failed`
- `alertkind`: `day_closed_pending_report`, `overdue_task_failed`
- `alertdispatchstatus`: `pending`, `dispatching`, `sent`
- `progresseventtype`: `daily_task.closed`, `daily_task.auto_failed`,
  `system_alert`, `custom_update`
- `progresspublishstatus`: `pending`, `publishing`, `published`, `failed`

Material tables and material enum values are intentionally absent after
`202605280002_drop_materials.sql`.

## Local E2E Control

The worker can expose control endpoints in non-production environments:

- `POST /control/reset?chat_id=...`
- `GET /control/topics?chat_id=...`
- `POST /control/clock`
- `POST /control/tick`

These endpoints make Telegram E2E deterministic for reset, time travel, worker
transitions, alert dispatch, and progress publishing. They are disabled when
`TRACKMATE__ENVIRONMENT=production`.

# Current Architecture

Trackmate 2.0 is a Go Telegram bot backed by PostgreSQL. It runs as two long-lived processes plus a one-shot migration command.

## Processes

- `cmd/migrate`: applies goose migrations from `migrations/`.
- `cmd/trackmate-api`: polls Telegram Bot API updates and routes them through `internal/bot`.
- `cmd/trackmate-worker`: runs periodic ticks for task transitions, alert dispatch, progress publishing, and non-production E2E control.
- `cmd/trackmate-healthcheck`: validates process health in Docker health checks.

## Runtime Boundaries

- `internal/config`: environment loading and production guards.
- `internal/logging`: structured slog setup.
- `internal/telegram`: typed Bot API client, update models, retry/error classification, and safe message helpers.
- `internal/dispatcher`: mailbox ordering by workspace/user so related updates are processed serially.
- `internal/bot`: explicit update router for setup, Today, reports, and alert acknowledgements.
- `internal/app/setup`: forum/admin prerequisite checks and Today/Progress topic repair.
- `internal/app/today`: daily task transition rules.
- `internal/app/progress`: progress outbox publishing.
- `internal/storage/postgres`: pgx storage, transactions, idempotency, DB claims, advisory worker lock, and E2E control state.
- `internal/ui`: Telegram HTML formatters and inline keyboards.
- `internal/control`: local-only reset/clock/tick/topics HTTP endpoints.

## Product Surface

Trackmate owns exactly two forum topics:

- `today`: title `Сегодня`, contains the control message, task cards, report prompts, and alerts.
- `progress`: title `Прогресс`, contains published progress events.

Materials is deleted from the product. The Go runtime does not create a Materials topic, parse Materials callbacks, store material rows, or publish material progress.

## Setup Flow

Setup can be triggered by `/setup`, bot membership updates, or setup buttons.

The bot checks:

- chat is a supergroup;
- forum topics are enabled;
- bot is admin or owner;
- bot can manage topics;
- bot can read participant messages.

When prerequisites pass, setup creates or repairs Today and Progress only. It stores topic bindings and setup/control/intro message IDs in PostgreSQL. Re-running setup is idempotent.

## Today Flow

`today:add` creates a pending `daily_task_text` input scoped to the Today thread. The next message from that user is accepted only if it arrives in the saved Today thread.

Task creation is protected by:

- one pending input per workspace/user;
- one daily task per participant per local day;
- a block on creating a new task while the previous task is still open.

Task cards stay in Today and include a report button while the task is open.

`task:report:<task_id>` opens a report status prompt. `task:status:<task_id>:<done|partial|failed>` stores pending `daily_task_report`. The next message in Today is claimed through `DELETE ... RETURNING`, updates the task card, and creates a `daily_task.closed` progress event.

Wrong-topic input is ignored without consuming the pending input.

## Worker Flow

Each tick takes a PostgreSQL advisory lock before state transitions.

Transitions:

- after local midnight: `active` to `awaiting_report`, plus `day_closed_pending_report` alert;
- after local noon: `active` or `awaiting_report` to `failed`, plus `overdue_task_failed` alert and `daily_task.auto_failed` progress event.

Alerts are claimed with `FOR UPDATE SKIP LOCKED`. Sent alert messages store their Telegram message ID. Acknowledgement marks `acknowledged_at` and clears the stored message ID after deleting the visible alert card.

Progress events are also claimed with `FOR UPDATE SKIP LOCKED`. Transient Telegram errors requeue; permanent failures mark the event failed.

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
- `progresseventtype`: `daily_task.closed`, `daily_task.auto_failed`, `system_alert`, `custom_update`
- `progresspublishstatus`: `pending`, `publishing`, `published`, `failed`

Material tables and material enum values are intentionally absent after migration `202605280002_drop_materials.sql`.

## Local E2E Control

The worker can expose control endpoints in non-production environments:

- `POST /control/reset?chat_id=...`
- `GET /control/topics?chat_id=...`
- `POST /control/clock`
- `POST /control/tick`

These endpoints make Telegram E2E deterministic for reset, time travel, worker transitions, alert dispatch, and progress publishing. They are disabled when `TRACKMATE__ENVIRONMENT=production`.

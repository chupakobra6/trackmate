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
- `internal/bot`: explicit update router for setup, Today, reports, routines,
  seasonal goals, and alert acknowledgements.
- `internal/app/setup`: forum/admin checks and product topic repair.
- `internal/app/today`: daily task rules and report state transitions.
- `internal/app/progress`: progress outbox formatting and publishing.
- `internal/app/routine`: routine check-in dispatch and leaderboard refresh.
- `internal/app/goals`: seasonal goal weekly/final dispatch and throttled goal
  nudges.
- `internal/storage/postgres`: pgx storage, transactions, idempotency, DB
  claims, advisory worker lock, and E2E control state.
- `internal/ui`: Telegram HTML formatters and inline keyboards.
- `internal/control`: local-only reset/clock/tick/topics HTTP endpoints.

## Product Surface

Trackmate owns four Telegram forum topics:

- `today`: title `Сегодня`; contains the control message, task cards, report
  prompts, and alerts.
- `routine`: title `Рутины`; contains routine setup, daily check-in cards, and
  the routine leaderboard.
- `goals`: title `Цели`; contains seasonal goal setup, weekly reviews, and final
  period reviews.
- `progress`: title `Прогресс`; contains published progress events.

Setup is idempotent. It creates or repairs only these topics, stores their
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

Wrong-topic daily task/report input is ignored without consuming pending state.

When Telegram sends `edited_message` for an already accepted user input,
Trackmate matches it by the stored source `message_id`/thread/user in
`daily_tasks`. Task text edits update the stored task, the Today card, and any
existing task progress payload. Report edits update the stored report, the Today
card, pending progress payloads, and already published Progress messages. This
path is silent: no additional Telegram messages are sent to the group.

## Routine Flow

`routine:configure` creates one pending `routine_plan` input scoped to the
Routines thread. The user sends a text list, one item per line. The parser
accepts plain lines, bullets, and numbered lists, and caps the list at 9 daily
items.

If the user starts a Routine/Goals setup draft and then switches to another
setup topic, Trackmate cancels the previous draft, removes the old bot prompt,
and removes the wrong-topic user message. This keeps unfinished setup input from
leaking across topics.

The worker creates one routine check-in card per participant per local day after
09:00, starting the morning after the plan was configured. The card is advanced
in place with `routine:item:<checkin_id>:<index>:<done|partial|failed>`.

`partial` and `failed` ask for one short reason. After all items are answered,
the same card asks for one reflection:

`Что помогло / что помешало / какую одну правку сделаешь завтра?`

Routine results stay in `Рутины`. They do not create `progress_events`.

The Routines topic also keeps a leaderboard message with 7-day completion rate,
current streak, best streak, and routine item count. Ranking uses completion
rate first, then current streak, so a one-item routine does not dominate by
streak alone.

## Goals Flow

`goals:configure` creates one pending `seasonal_goals` input scoped to the Goals
thread. Goals are stored as raw Telegram HTML for the current season. The setup
confirmation is intentionally short and does not echo the full goals text back
into the topic. The instruction asks for a measurable format:

- `Результат`
- `Метрика`
- `Еженедельный шаг`
- `Почему важно`

The first live period is `Лето 2026`, ending on `2026-09-01`; the period helper
then follows calendar seasons.

On Sunday after 20:00 local time, the worker sends one weekly review prompt in
`Цели` and stores the response as `goal_weekly_review`. On and after the period
end date, the worker sends a final review prompt with buttons
`done|partial|failed`; after the button, the user writes one final summary.

Today can show a rare deterministic goal nudge when a participant already has
seasonal goals for the current period. Nudges are pseudo-random by seed, but
persist a per-user cooldown in PostgreSQL and cannot appear more than once every
72 hours.

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

The same tick also dispatches due routine check-ins, weekly goal reviews, and
final goal reviews. These are idempotent through stored Telegram message IDs and
do not pass through the Progress outbox.

## Data Model

Core tables:

- `workspace_groups`
- `topic_bindings`
- `participants`
- `daily_tasks`
- `daily_task_alerts`
- `pending_inputs`
- `progress_events`
- `routine_plans`
- `routine_checkins`
- `routine_checkin_items`
- `seasonal_goal_sets`
- `seasonal_goal_weekly_reviews`
- `seasonal_goal_final_reviews`
- `goal_nudge_cooldowns`
- `app_clock`

Important enum values:

- `topickey`: `today`, `progress`, `routine`, `goals`
- `dailytaskstatus`: `active`, `awaiting_report`, `done`, `partial`, `failed`
- `routineitemstatus`: `done`, `partial`, `failed`
- `goalfinalstatus`: `done`, `partial`, `failed`
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

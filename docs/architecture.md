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
- `internal/bot`: explicit update router for setup, Today, reports, routines,
  seasonal goals, and alert acknowledgements.
- `internal/app/setup`: forum/admin checks and product topic repair.
- `internal/app/today`: daily task rules and report state transitions.
- `internal/app/progress`: progress outbox formatting and publishing.
- `internal/app/routine`: routine check-in dispatch and leaderboard refresh.
- `internal/app/goals`: seasonal goal review/final dispatch and throttled goal
  nudges.
- `internal/storage/postgres`: pgx storage, transactions, idempotency, DB
  claims, advisory worker lock, and E2E control state.
- `internal/ui`: Telegram HTML formatters and inline keyboards.
- `internal/control`: local-only reset/clock/tick/topics HTTP endpoints.

## Delivery And Concurrency

The API handles Telegram updates in the order returned by `getUpdates`. The
polling offset advances only after an update handler succeeds. A failed update
therefore remains unacknowledged and is retried with backoff; already completed
updates before it form the acknowledged prefix. Trackmate intentionally has no
second in-memory dispatch queue, so process shutdown cannot strand fetched work
behind an already advanced cursor.

The worker serializes ticks with a schema-namespaced PostgreSQL session
advisory lock. The lock owns one acquired `pgxpool.Conn` from
`pg_try_advisory_lock` through
`pg_advisory_unlock`; releasing it uses a bounded background context so a
cancelled tick cannot return a locked session to the pool.

Alert and Progress outbox claims persist their start time. A process may
reclaim `dispatching` or `publishing` rows after five minutes, including legacy
rows with no start time. Completion and requeue operations require the claimed
state and report a lost claim instead of silently succeeding. Telegram sends
still cross an external-system boundary, so when the following database write
fails, worker flows make a bounded attempt to delete the just-sent message
before making the row retryable.

## Product Surface

Trackmate owns four Telegram forum topics:

- `today`: title `Сегодня`; contains the control message, task cards, report
  prompts, and alerts.
- `routine`: title `Рутины`; contains routine setup, daily check-in cards, and
  the routine leaderboard.
- `goals`: title `Цели`; contains seasonal goal setup, reviews every two weeks, and final
  period reviews.
- `progress`: title `Прогресс`; contains published progress events.

Setup is idempotent. It creates or repairs only these topics, stores their
thread IDs and message IDs in PostgreSQL, and does not create a Materials topic.

Materials is deleted from runtime and schema. The bot does not parse
`material:*` callbacks as product actions, does not store material rows, and does
not publish material progress.

## Today Flow

`today:add` is local-time aware. Before 20:00 it creates one pending
`daily_task_text` input scoped to the Today thread. The next message from that
user is accepted only if it arrives in that thread.

At or after 20:00, when the participant has no entry for the local date, the
same button creates an open `summary` entry instead of a retrospective task.
Its card asks for one of three outcomes: `Хорошо`, `Средне`, or `Плохо`.

Daily entry creation is protected by:

- one pending input per workspace/user/thread;
- one entry per participant per local day: either a task or a summary;
- a block on creating a new task while the previous regular task is still open.

Task cards stay in Today and include a report button while the task is open.
Summary cards stay in Today and show their three status buttons while open.

`task:report:<task_id>` opens the report flow by editing the pressed task card
or alert in place into the status chooser. Replayed callbacks edit that same
message and never send another chooser, so repeated taps cannot stack duplicate
`Выбери итог дня` messages.
`task:status:<task_id>:<done|partial|failed>` stores pending
`daily_task_report` or `daily_summary_report`. The next Today message is
claimed through the database, updates the card, and creates a
`daily_task.closed` or `daily_summary.closed` progress event. Summary status
`partial` is rendered as `Средне`; it is never shown as `Частично`.

Alert and notice dismissal first tries to delete the Telegram message. If an
older message can no longer be deleted, Trackmate replaces it with an inert
`Уведомление закрыто` tombstone and no buttons; keyboard removal is the final
fallback. An alert is acknowledged and detached from its Telegram message in
the database only after one of those UI transitions succeeds. The callback
message ID is retained as a recovery path for alerts affected by older versions
that cleared the stored message ID too early.

Wrong-topic daily task/report input is ignored without consuming pending state.

When Telegram sends `edited_message` for an already accepted user input,
Trackmate matches it by the stored source `message_id`/thread/user in
`daily_tasks`. Task text edits update the stored task, the Today card, and any
existing task progress payload. Task and summary report edits update the stored
report, the Today card, pending progress payloads, and already published
Progress messages. On the normal path this stays silent. If Telegram refuses to
edit an old bot message, Trackmate preserves the database update and queues a
`system_alert` in `Прогресс` with a link to the message and the Telegram error.

## Routine Flow

`routine:configure` creates one pending `routine_plan` input scoped to the
Routines thread. The user sends a text list, one item per line. The parser
accepts lines that start with `-`, `—`, `1.`, or `1)`, and caps the list at 9
daily items. Plain lines and unsupported bullet symbols are rejected so the setup
format stays unambiguous.

Routine and Goals setup share one recoverable prompt lifecycle. Repeated
configure callbacks verify and reuse the stored prompt instead of sending a
second copy. A replacement is sent and attached to the existing pending input
only when Telegram explicitly reports that the stored message no longer
exists; transient edit failures are returned without creating a duplicate.

Pending input is isolated by Telegram topic thread. A Routine draft does not
block Today or Goals, and a message from another thread does not consume or
cancel the Routine draft. Worker cleanup removes pending inputs older than 24
hours, deleting the stored bot prompt and known process messages silently.

The worker creates one routine check-in card per participant after 08:00 local
time for the previous local day. A plan created today can first produce a card
tomorrow morning for today's routine. The card is advanced in place with
`routine:item:<checkin_id>:<index>:<done|partial|failed>`.

`partial` and `failed` mark the item in the main card and send a separate short
reason prompt. After the user answers, Trackmate deletes that prompt and the
user reply, stores the reason, and advances the main card. After all items are
answered, the routine check-in closes immediately without a separate day
reflection.

Routine results stay in `Рутины`. They do not create `progress_events`.

If a routine card is still open at 20:00 local time on the check-in day,
Trackmate sends a single reminder in `Рутины`. At 00:00 the next local day,
missing items are marked as `failed` and the existing check-in card is edited
in place to its final state without buttons. A short auto-close alert names the
participant, links the source routine, and is posted as a reply to that card.
If the card was removed outside Trackmate, the source routine message becomes
the reply fallback. Failed Telegram delivery remains eligible for a later
worker retry; a delivered alert is not recreated after its temporary message
is dismissed or cleaned up. Reminder and auto-close alerts are cleaned up after
about 24 hours if the user does not dismiss them first.

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

Every second Sunday after 20:00 local time, the worker sends one goals review
prompt in `Цели` and stores the response as `goal_weekly_review`. On and after
the period end date, the worker sends a final review prompt with buttons
`done|partial|failed`; after the button, the user writes one final summary.
An unanswered two-week review has its own persisted lifecycle: after 24 hours
the first prompt is removed and sent once again; 72 hours after the first
successful delivery, the retry is removed and the review is stored as skipped.
The generic pending-input cleanup does not own these prompts, so the retry stays
answerable until the shared 72-hour deadline. A final period review waits until
an open two-week review is answered or skipped.

Today can show a rare deterministic goal nudge when a participant already has
seasonal goals for the current period. Nudges are pseudo-random by seed, but
persist a per-user cooldown in PostgreSQL and cannot appear more than once every
72 hours.

## Worker Flow

Each tick takes a PostgreSQL advisory lock before transitions.

Transitions for both task and summary entries:

- after local midnight: `active` to `awaiting_report`, plus
  `day_closed_pending_report` alert;
- after local noon: `active` or `awaiting_report` to `failed`, plus
  `overdue_task_failed` alert and a `daily_task.auto_failed` or
  `daily_summary.auto_failed` progress event.

Alerts and progress events are claimed with `FOR UPDATE SKIP LOCKED`. Telegram
transient failures are requeued; permanent failures are marked failed.

Alert acknowledgement deletes the visible alert card, marks `acknowledged_at`,
and clears the stored Telegram message ID.

The same tick also dispatches due routine check-ins, goals reviews, and
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
  - one active input per `workspace_group_id`, `user_id`, `message_thread_id`
  - stale inputs older than 24 hours are removed by the worker
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
- `dailyentrykind`: `task`, `summary`
- `routineitemstatus`: `done`, `partial`, `failed`
- `goalfinalstatus`: `done`, `partial`, `failed`
- `alertkind`: `day_closed_pending_report`, `overdue_task_failed`
- `alertdispatchstatus`: `pending`, `dispatching`, `sent`
- `progresseventtype`: `daily_task.closed`, `daily_task.auto_failed`,
  `daily_summary.closed`, `daily_summary.auto_failed`, `system_alert`,
  `custom_update`
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

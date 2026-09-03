# Handoff

Проект: trackmate
Обновлено: 2026-09-04

## Цель
- Восстановить пропущенный финал летних целей и убрать failure mode, который блокировал сезонный переход.

## Завершенный Шаг
- step: `STEP-038`
- status: `готово`
- requirements: `REQ-063..REQ-066`, `VAL-016`
- source: `S039`

## Root Cause
- Summer boundaries in production were correct: `2026-06-01..2026-09-01`.
- Weekly retry was sent after 24h and skipped after 72h from the first prompt, exactly 48h after retry delivery.
- Telegram rejected delete at that boundary; `advanceWeeklyReview` returned before persisting `skipped_at`.
- The sequential worker tick therefore never reached final goals, alert delivery, or progress publishing.
- Igor/Yaroslav summaries were consumed by old weekly pending inputs; Igor's second message had no pending and was not stored.
- Goal setup/control copy also hardcoded `Лето 2026`.

## Product Fix
- Weekly close is due at 71h and persists skip before best-effort delete/inert edit.
- Stored season boundaries use calendar-date semantics across timezones.
- Goal setup renders the current local season dynamically; the pinned control has no stale date.
- Final reflection accumulates idempotent source messages until owner-only `✅ Завершить итог`.
- Weekly/final cards link to source responses instead of echoing unbounded text.
- Final goal pending inputs are excluded from generic 24h cleanup.

## Validation
- DB-backed `make check`: pass.
- migration `202609040001`: local/prod pass and columns read back.
- PostgreSQL bot integration covers two long final messages, idempotent persistence, explicit completion, source links and edit fallback.
- focused Telegram E2E: not started because runner MTProto session requires re-login; no E2E step mutated the test group.

## Production Deploy
- target: `inferno-nl:/opt/trackmate`.
- code: `087ad1c -> 7059d88`.
- backup: `/opt/trackmate/backups/trackmate_20260903T223606Z.dump`; checksum and `pg_restore --list` pass; `245029` bytes; mode `600`.
- schema: `202609040001`.
- final reviews:
  - goal set `1`, owner `1266978055`, status `partial`, parts `{7207,7208}`, card `6686`;
  - goal set `4`, owner `758888600`, status `partial`, parts `{7225}`, card `6687`;
  - goal set `6`, owner `1747674822`, prompt `7248`, incomplete and waiting for Egor.
- weekly rows `11,12,13` are skipped and no longer contain misclassified final text; pending `874` removed.
- Telegram/Harvest verifies card links `6686 -> 7207/7208`, `6687 -> 7225`, inert `6688`, and Egor prompt `7248 -> 4192`.
- stale pending alerts `150,151,163,164` acknowledged before restart; current `165` delivered.
- all `17` blocked progress events published. Temporary Telegram `429` warnings occurred during backlog drain, then stopped; no data remained pending.
- final health: all services healthy; weekly-open `0`; unpublished alerts/progress `0`; stale claims `0`; idle transactions `0`; advisory waiters `0`; fresh errors/warnings `0`.

## User-Visible Copy Changes
- Setup prompt: `Текущий период: <b>Лето 2026</b> (до <b>01.09.2026</b>)` -> dynamic `{{period}}` / `{{date}}` (currently Autumn 2026 / 01.12.2026).
- Pinned Goals control no longer shows a hardcoded summer/date line.
- `Опиши конкретные результаты по целям:` -> `Опиши конкретные результаты одним или несколькими сообщениями:`.
- Added `✅ Завершить итог`, `Сохранено частей: {{count}}`, completion hint, source-part links, and inert `⌛ Эта проверка целей закрыта`.
- Weekly saved card no longer echoes the response; it shows `Открыть ответ`.

## Следующее Действие
- Егор completes the summer final from prompt `7248`; otherwise no operational follow-up is required.

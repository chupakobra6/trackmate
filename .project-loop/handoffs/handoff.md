# Handoff

Проект: trackmate
Обновлено: 2026-08-24

## Цель
- Защитить routine reminder/auto-close от чужого `Понял` и развернуть общий callback access fix.

## Завершенный Шаг
- step: `STEP-037`
- status: `готово`
- requirements: `REQ-062`, `VAL-015`
- source: `S038`

## Проверка
- `TestRunCheckinTransitionsRemindsAndAutoCloses`: pass; обе routine-кнопки имеют `notice:dismiss:42`.
- `TestPersonalCallbacksRejectOtherParticipantsWithoutMutation`: pass; user `43` не удаляет notice владельца `42`, Telegram/БД не меняются.
- `TestNoticeDismissDeletesMessage`: pass; владелец закрывает свой notice.
- `TestOwnerlessDismissIsRejectedWithoutDeletingMessage`: pass.
- DB-backed `make check`: pass; полный Telegram E2E не запускался.

## Production Deploy
- target: `inferno-nl:/opt/trackmate`
- before: `a9ad102`, clean worktree, healthy services.
- pushed/deployed: `ec6c31b`; product fix commit `03a4c78`.
- backup: `/opt/trackmate/backups/trackmate_20260824T114142Z.dump` (225K); checksum, metadata и `pg_restore --list` pass.
- after: `api`, `worker`, `postgres` healthy; migration job success; schema `202608050002`.
- stale alert claims `0`; stale progress claims `0`; unpublished progress `0`; idle transactions `0`; advisory holders/waiters `0/0`; fresh suspicious logs `0`.
- `2` historical sent/unacknowledged task alerts remain unchanged and are protected through persisted task ownership.
- active pending inputs `3` were not mutated.

## Existing Routine Alert Repair
- Production had one active old-version auto-close notice: checkin `1842455`, message `6641`, owner `1747674822`.
- Its reply markup was updated through Bot API to `notice:dismiss:1747674822`; response `ok`, message id matched `6641`.
- No active routine reminder existed; no other routine alert required repair.

## Review
- Routine generation and bot authorization are connected by the same exact callback contract.
- New callback kinds remain deny-by-default.
- No database rows or routine results were manually changed; only the known active message markup was repaired.

## Следующее Действие
- Обычное наблюдение; новые routine alerts уже создаются с owner-bound callbacks.

## Обновленные Источники Правды
- `.project-loop/requirements/source-map.md`
- `.project-loop/requirements/checklist.md`
- `.project-loop/plan/delivery-plan.md`
- `.project-loop/plan/current-step.md`
- `.project-loop/intake/user-deltas.md`
- `.project-loop/handoffs/handoff.md`

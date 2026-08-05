# Текущий Шаг

Проект: trackmate
Обновлено: 2026-08-05

## Активный Шаг
- id: `STEP-035`
- status: `в работе`
- objective: Упростить delivery concurrency и устранить потерю Telegram updates, session-leak advisory lock и навсегда зависающие delivery claims без изменения пользовательского вида.
- requirement IDs: `REQ-059`, `REQ-060`, `VAL-013`
- owned paths: `cmd/trackmate-api/`, `internal/dispatcher/`, `internal/worker/`, `internal/app/progress/`, `internal/app/routine/`, `internal/app/goals/`, `internal/storage/postgres/`, additive migration, architecture/operations docs, `AGENTS.md`, `.project-loop/`.
- validation: focused unit/PostgreSQL concurrency tests; additive migration; focused worker/app packages; full Go suite, lint/vet; только затронутые Telegram scenarios; production backup/deploy/read-only verification.
- done criteria: success-based Telegram offset; no in-memory unacknowledged queue; same-session worker lock; crash-recoverable alert/progress claims; compensated worker send/persist failures; clean production services/queues/logs.

## Фокус Ревью
- Не оставлять compatibility aliases или параллельные старые contracts.
- Удалить dispatcher, если последовательная обработка сохраняет нужный пользовательский flow и делает cursor ownership однозначным.
- Проверить границы внешнего Telegram send и DB persistence; exactly-once недоступен, поэтому нужны persisted lease + безопасная компенсация.
- Не менять тексты, кнопки, расписания или продуктовые состояния.

## Проверенная Исходная Картина
- Production на `76db6c2`, сервисы healthy; alerts: `127 sent`, progress: `268 published`, stuck rows не обнаружены.
- `pg_try_advisory_lock` и `pg_advisory_unlock` вызываются через pool без закрепленной session.
- `dispatching`/`publishing` не имеют времени claim и после crash не восстанавливаются.
- Telegram offset увеличивается до завершения async handler, поэтому следующий poll может подтвердить еще не обработанный update.
- Полный Telegram E2E не запускается по явному пользовательскому правилу.

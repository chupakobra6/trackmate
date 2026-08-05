# Текущий Шаг

Проект: trackmate
Обновлено: 2026-08-05

## Завершенный Шаг
- id: `STEP-035`
- status: `готово`
- objective: Упростить delivery concurrency и устранить потерю Telegram updates, session-leak advisory lock и навсегда зависающие delivery claims без изменения пользовательского вида.
- requirement IDs: `REQ-059`, `REQ-060`, `VAL-013`
- delivered commit: `ee07db3`
- production: `ee07db3`, migrations `202608050001` и `202608050002` applied.
- backup: `/opt/trackmate/backups/trackmate_20260805T113344Z.dump` (checksum + archive listing pass).

## Итог
- Telegram updates обрабатываются последовательно; offset двигается только после успешного handler, async mailbox dispatcher удален.
- Worker lock schema-namespaced и живет на одном pinned `pgxpool.Conn` до bounded background unlock.
- Alert/Progress claims имеют persisted 5-minute lease, legacy/crash reclaim и проверяемые terminal transitions.
- Worker send-then-persist paths компенсируют ошибку bounded удалением только что отправленного Telegram message.
- Инварианты сохранены в `AGENTS.md`/architecture/operations; `make check` объединяет lint, vet и tests.

## Валидация
- Clean-schema PostgreSQL concurrency/delivery tests: pass.
- `TRACKMATE_TEST_DATABASE_URL=... make check`: pass.
- Local migration/readback, Docker health/log scan: pass.
- Focused live alert/worker smoke: task `134`, alert `21`, message `860`, dismiss pass; test data/history cleaned.
- Full Telegram E2E намеренно не запускался.
- Production: services healthy; stale alert/progress claims `0`; idle transactions `0`; worker advisory waiters `0`; fresh error/warn scan empty.

## Сохраненное Production-Состояние
- Один свежий `seasonal_goals` pending Игоря оставлен без изменений как активный ввод.
- Два исторических sent/unacknowledged alerts за май оставлены без ручной мутации; они не claimable и не создают повторных отправок, а новый callback lifecycle умеет закрыть их при клике.

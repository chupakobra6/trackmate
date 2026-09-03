# Текущий Шаг

Проект: trackmate
Обновлено: 2026-09-04

## Завершенный Шаг
- id: `STEP-040`
- status: `готово`
- objective: Провести полный аудит Telegram deletion window и закрыть класс граничных удалений.
- requirement IDs: `REQ-069`, `REQ-070`, `VAL-018`
- source IDs: `S042`, `S043`

## Подтверждённый Контракт
- Официальный Bot API разрешает `deleteMessage` только для сообщений младше `48h`.
- Trackmate использует `1h` safety margin: максимальный плановый возраст удаления — `47h`.
- Weekly `71h` = initial-to-reminder `24h` + retry safe age `47h`.
- `GoalNudgeCooldown=72h` не связан с удалением и не должен механически меняться.

## Результат
- Все scheduled application delete paths используют единый age-aware `internal/app/messagecleanup` contract.
- Жёсткая граница — `<48h`; плановый target — `47h`; weekly deadline вычисляется как `24h + 47h = 71h`.
- Просроченное или неудаляемое bot message переводится в inert state, а worker lifecycle продолжается.
- Architecture test запрещает новые прямые `DeleteMessage` call sites в application code, кроме just-sent delivery compensation.
- Интерактивные bot cleanup paths классифицированы как immediate/user-triggered и остаются non-blocking.
- Каноническая production backup-команда зафиксирована после безопасно обработанного `Permission denied` прямого запуска скрипта.

## Валидация
- focused tests, fresh `go test ./... -count=1`, `make check`, `git diff --check`: pass.
- Project Loop validation: pass.
- commit `c1e126d` в local/origin/production; backup проверен.
- production services healthy; все queue/stale/deadline/waiter counters `0`; fresh error scan clean.

## Следующее Действие
- Обязательных действий нет; invariant и regression guard защищают будущие изменения.

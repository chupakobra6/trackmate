# Текущий Шаг

Проект: trackmate
Обновлено: 2026-09-04

## Активный Шаг
- id: `STEP-040`
- status: `в работе`
- objective: Провести полный аудит Telegram deletion window и закрыть класс граничных удалений.
- requirement IDs: `REQ-069`, `REQ-070`, `VAL-018`
- source IDs: `S042`, `S043`

## Подтверждённый Контракт
- Официальный Bot API разрешает `deleteMessage` только для сообщений младше `48h`.
- Trackmate использует `1h` safety margin: максимальный плановый возраст удаления — `47h`.
- Weekly `71h` = initial-to-reminder `24h` + retry safe age `47h`.
- `GoalNudgeCooldown=72h` не связан с удалением и не должен механически меняться.

## Область
- все production `DeleteMessage` call sites;
- domain cleanup/retry durations;
- worker delete-failure behavior;
- regression tests и `AGENTS.md` invariant;
- commit, push, production deploy и live verification.

## Критерий Готовности
- scheduled delete paths планируются не позже safe age `47h` и никогда не вызывают Telegram deletion на/после hard limit `48h`;
- delayed worker делает сообщение inert или безопасно закрывает lifecycle без повторного loop;
- все удаления классифицированы и покрыты проверяемым контрактом;
- DB-backed checks и production verification проходят.

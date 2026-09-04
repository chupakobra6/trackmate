# Текущий Шаг

Проект: trackmate
Обновлено: 2026-09-04

## Активный Шаг
- id: `STEP-042`
- status: `в работе`
- objective: Исправить streak рутины, поздний результат auto-failed задачи и последнюю задачу Игоря.
- requirement IDs: `REQ-073`, `REQ-074`, `REQ-075`, `VAL-020`
- source IDs: `S045`

## Подтверждённый Контракт
- Полный день рутины увеличивает серию; partial сохраняет накопленную серию без увеличения; failed и календарный пропуск сбрасывают.
- До завершения текущей routine-проверки накопленная серия не обнуляется.
- Auto-fail в 12:00 редактирует исходную task card в failed-состояние без кнопки.
- Временный overdue alert сохраняет действия `Результат` и `Понял`.
- Один поздний результат можно записать через alert; задача остаётся `failed`, а существующее auto-fail Progress событие дополняется результатом без дубля.
- Последняя задача Игоря восстанавливается как `done` только по точному production evidence уже отправленного своевременного результата.

## План Реализации
1. Сверить production DB/logs/Telegram для последней задачи Игоря и не менять данные до точной идентификации.
2. Исправить streak-классификацию и добавить проверки full/partial/failed/open/gap.
3. Исправить worker card finalization и late-report storage/bot/progress lifecycle с регрессионными тестами.
4. Обновить публичную архитектурную документацию и focused E2E expectation, если контракт там описан.
5. Пройти focused и broad проверки; сделать path-specific commit и push.
6. Снять production backup, развернуть revision, восстановить одну задачу Игоря и проверить DB/Telegram/services/queues/logs.

## Валидация
- focused tests для storage/bot/worker/UI;
- DB-backed `make check`, fresh `go test ./... -count=1`, `git diff --check`;
- Project Loop validation;
- production revision/backup/schema, affected task/progress/card readback, service health, queues/claims и fresh logs.

## Следующее Действие
- Получить exact production evidence и реализовать минимальный root-cause fix.

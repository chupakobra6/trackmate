# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-013`
- status: `готово`
- objective: Переделать routine reason flow на отдельное временное сообщение и убрать финальный итог дня из рутин.
- requirement IDs: `REQ-030`
- owned paths: `.project-loop/`, `internal/domain/types.go`, `internal/bot/routines.go`, `internal/bot/service.go`, `internal/bot/service_integration_test.go`, `internal/storage/postgres/routines.go`, `internal/storage/postgres/storage_integration_test.go`, `internal/ui/formatters.go`, `internal/ui/formatters_test.go`, `internal/ui/keyboards.go`, `internal/app/routine/routine_integration_test.go`, `docs/`, `e2e/telegram/scenarios/12-routine-checkin.jsonl.tmpl`
- validation: `go test ./internal/ui ./internal/bot ./internal/app/routine ./internal/storage/postgres`: pass; `go test ./... -count=1`: pass; `make lint`: pass; `loopctl.py validate`: pass
- done criteria: production факт проверен read-only; `Нет`/`Частично` отправляют отдельный reason prompt reply к основной карточке; после ответа prompt и user reply удаляются; основная карточка не превращается в prompt причины; после всех пунктов check-in закрывается без routine final reflection; prod deploy не выполнялся.

## Фокус Ревью
- Менять только routine flow из текущего скриншота; цели не трогать без отдельного конкретного сценария.
- Не чистить production данные и не выкатывать фикс до отдельной пачки.
- Сохранить хранение причин и таблицу рутин, не публиковать routine events в `Прогресс`.

## Примечания
- Production logs 2026-06-24 17:00 UTC показали: worker отправил routine cards `3373`/`3374`; callback `routine:item:287:0:failed` пришел по message `3373`, затем user message `3375`; текущий production flow редактирует основную карточку для причины.
- S012 заменяет часть раннего MVP: routine final reflection больше не нужна, потому что итог дня живет в `Сегодня`.

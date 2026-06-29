# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-012`
- status: `готово`
- objective: Унифицировать prompt причины рутины с обычной карточкой и заменить первый emoji routine header.
- requirement IDs: `REQ-029`
- owned paths: `.project-loop/`, `internal/ui/formatters.go`, `internal/ui/formatters_test.go`, `internal/bot/routines.go`, `internal/bot/service_integration_test.go`
- validation: `go test ./internal/ui ./internal/bot`: pass; `go test ./... -count=1`: pass; `make lint`: pass; `loopctl.py validate`: pass
- done criteria: production факт проверен read-only; reason prompt использует тот же заголовок с автором, что и карточка; вопрос остается в формате `N/M: пункт?`; строка `Что помешало?` добавлена без отдельного визуального шаблона; первый routine emoji заменен; prod deploy не выполнялся.

## Фокус Ревью
- Не менять routine logic, storage, расписание и auto-close.
- Не чистить production данные и не выкатывать фикс до отдельной пачки.
- Исправлять только проблему со скриншота.

## Примечания
- Production logs 2026-06-24 17:00 UTC показали: обычные карточки `3373`/`3374`, затем callback по `3373` и edit в reason prompt.
- Причина разного вида: `FormatRoutineReasonPrompt` строил текст отдельно от `FormatRoutineCheckinCard` и не получал `displayName/username`.

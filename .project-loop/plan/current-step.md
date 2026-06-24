# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-24

## Активный Шаг
- id: `STEP-011`
- status: `готово`
- objective: Уточнить текст карточки рутины, чтобы дата читалась как день, за который нужно отметить пункты.
- requirement IDs: `REQ-028`
- owned paths: `.project-loop/`, `internal/ui/formatters.go`, `internal/ui/formatters_test.go`
- validation: `go test ./internal/ui ./internal/bot ./internal/app/routine`: pass; `make lint`: pass; `loopctl.py validate`: pass
- done criteria: карточка рутины содержит `Рутина за DD.MM`; внутри есть короткая строка о том, что отмечаются пункты за этот день; prompt причины сохраняет тот же смысл; тесты проходят.

## Фокус Ревью
- Карточка должна быть понятна утром/вечером и при просмотре на следующий день.
- Текст остается коротким и без лишнего объяснительного полотна.

## Примечания
- На текущем production `24.06` означает routine check-in за `24.06`, а не за `23.06`.
- В локальной ветке STEP-010 уже перевел routine check-in на вечерний сценарий; STEP-011 только уточняет copy карточки.

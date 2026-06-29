# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-015`
- status: `готово`
- objective: Централизовать тексты в `messages.md`, упростить routine copy/input и добавить dismiss для служебных problem messages.
- requirement IDs: `REQ-032..REQ-034`
- owned paths: `.project-loop/`, `internal/messages/`, `internal/ui/`, `internal/bot/`, `internal/app/`, `internal/domain/`, `internal/telegram/`, `e2e/telegram/scenarios/`
- validation: `go test ./internal/messages ./internal/domain ./internal/ui ./internal/telegram ./internal/app/goals ./internal/app/routine ./internal/bot ./internal/worker`: pass; `make test`: pass; `make lint`: pass; `git diff --check`: pass; `loopctl.py validate .`: pass
- done criteria: visible bot copy is imported from one editable document; routine prompt uses dash examples and no max-count copy; routine parser accepts only `-`/`—` items; service notices/reminders have dismiss/action cleanup; tests pass.

## Фокус Ревью
- Сохранить поведение топиков, менять только copy/input/cleanup mechanics из S014.
- Не выносить runtime config или domain enums в copy-файл; только user-facing text.
- Не деплоить production до отдельной команды.

## Примечания
- S014 заменяет прежний широкий routine parser из S001.
- `messages.md` должен быть удобен для редакторского review: один файл с ключами и HTML-разметкой Telegram.
- Production не трогался в STEP-015.

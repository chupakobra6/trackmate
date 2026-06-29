# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-024`
- status: `готово`
- objective: Локально добавить персональный 30% текст для алертов Егора.
- requirement IDs: `REQ-045`
- owned paths: `internal/domain/`, `internal/app/routine/`, `internal/worker/`, `internal/ui/`, `internal/messages/`, `.project-loop/`
- validation: focused Go tests: pass; `go test ./...`: pass; `make test`: pass; `make lint`: pass; `git diff --check`: pass; `loopctl.py validate`: pass
- done criteria: routine reminder, routine auto-close notice and daily missed-task alert can use personalized Trackmate-style copy for Egor with deterministic 30% chance; non-target users keep the default copy.

## Фокус Ревью
- Это локальная продуктовая правка, не production deploy.
- Не делать Telegram cleanup и не трогать production DB.
- Сохранять тексты в `internal/messages/messages.md`, не размазывать новые русские строки по Go-коду.

## Примечания
- Персональный текст должен быть стабильным для одного события, чтобы retry не менял сообщение.
- Target ограничен username на `w` и display name с `Егор`, чтобы случайно не обращаться к другому `w...` пользователю как к Егору.
- Все новые видимые строки должны оставаться в `internal/messages/messages.md`.

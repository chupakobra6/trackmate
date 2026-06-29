# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-026`
- status: `готово`
- objective: Локально исправить сохранение целей: удалять prompt и отправлять новое подтверждение после ввода.
- requirement IDs: `REQ-048`
- owned paths: `internal/bot/`, `internal/ui/`, `internal/messages/`, `.project-loop/`
- validation: focused Go tests: pass; `go test ./...`: pass; `make test`: pass; `make lint`: pass; `git diff --check`: pass; `loopctl.py validate`: pass
- done criteria: goals input remains visible as the source message; the old prompt is deleted; confirmation is a new silent message after the user's input, does not reply/quote, links `Цели` to the source message, and does not echo the full goals text.

## Фокус Ревью
- Это локальная продуктовая правка, не production deploy.
- Не делать Telegram cleanup и не трогать production DB.
- Сохранять тексты в `internal/messages/messages.md`, не размазывать новые русские строки по Go-коду.

## Примечания
- Текущая topic isolation остается как в S009: разные топики не задевают pending inputs друг друга.
- При goals save пользовательское сообщение не удаляется, потому что оно источник для ссылки.
- Старый prompt Trackmate удаляется, а подтверждение отправляется отдельным сообщением после ввода, чтобы не ломать хронологию топика.

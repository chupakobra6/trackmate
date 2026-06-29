# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-017`
- status: `готово`
- objective: Поправить routine reminder/autoclose alerts: лаконичный текст, кнопка `Понял`, удаление при закрытии и TTL около суток.
- requirement IDs: `REQ-034,REQ-035,REQ-036`
- owned paths: `.project-loop/`, `internal/messages/`, `internal/domain/`, `internal/storage/postgres/`, `internal/app/routine/`, `internal/bot/`, `internal/worker/`, `internal/ui/`, `migrations/`
- validation: `go test ./internal/storage/postgres ./internal/app/routine ./internal/ui ./internal/domain`: pass; `go test ./internal/bot ./internal/worker ./internal/messages`: pass; `make test`: pass; `make lint`: pass; `git diff --check`: pass; `loopctl.py validate .`: pass
- done criteria: routine reminder text is short and clear; reminder has `Понял` and disappears on routine close or cleanup TTL; auto-close sends a short temporary notice; cleanup removes reminder/auto-close notices after about 24h; tests pass.

## Фокус Ревью
- Сохранить `Рутины` как чистый topic: постоянная таблица плюс временные рабочие/alert сообщения.
- Не возвращать старую тяжелую формулировку reminder: без `еще не закрыта` и `будут засчитаны`.
- Не деплоить production до отдельной команды.

## Примечания
- S015 заменяет часть S014: нумерация снова принимается, confirmation настройки рутины не остается.
- S016 заменяет часть S015: auto-close notice возвращен, но как временный dismissable alert с TTL.
- Production не трогался в STEP-017.

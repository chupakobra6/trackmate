# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-016`
- status: `готово`
- objective: Почистить routine topic flow: убрать save confirmation, удалить routine cards после ответа/автозакрытия и вернуть поддержку нумерации.
- requirement IDs: `REQ-033,REQ-035`
- owned paths: `.project-loop/`, `internal/messages/`, `internal/domain/`, `internal/bot/`, `internal/app/routine/`, `internal/ui/`, `e2e/telegram/scenarios/`
- validation: `go test ./internal/domain ./internal/ui ./internal/bot ./internal/app/routine ./internal/messages`: pass; `make test`: pass; `make lint`: pass; `git diff --check`: pass; `loopctl.py validate .`: pass
- done criteria: routine setup prompt/user input are deleted without persistent confirmation; numbered routine input is accepted; completed routine card is deleted after final answer; auto-close deletes card/reminder/pending prompt without sending notice; leaderboard still refreshes; tests pass.

## Фокус Ревью
- Сохранить `Рутины` как чистый topic: постоянная таблица плюс временные рабочие сообщения.
- Не возвращать свободный parser: нумерация поддерживается только как префикс пункта, остальные свободные строки не принимаются.
- Не деплоить production до отдельной команды.

## Примечания
- S015 заменяет часть S014: нумерация снова принимается, confirmation настройки рутины не остается.
- Production не трогался в STEP-016.

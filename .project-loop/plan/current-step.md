# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-24

## Активный Шаг
- id: `STEP-010`
- status: `готово`
- objective: Переделать pending inputs на изоляцию по топикам, добавить тихую очистку stale pending старше суток и перевести routine check-in на вечерний flow с напоминанием и автозакрытием.
- requirement IDs: `REQ-025..REQ-027`, `VAL-006`
- owned paths: `.project-loop/`, `migrations/`, `internal/domain/`, `internal/storage/postgres/`, `internal/bot/`, `internal/app/`, `internal/worker/`, `internal/ui/`, `docs/`, tests
- validation: focused Go tests for pending/routine worker behavior; `go test ./... -count=1`; `make test`; `make lint`; `loopctl.py validate`
- done criteria: pending input уникален по `workspace/user/thread`; сообщения и callbacks не блокируются pending из другого топика; stale cleanup удаляет старые prompt/user messages молча; routine card приходит вечером в день настройки при создании до вечернего времени; незакрытая routine получает напоминание после конца дня и автозакрывается в 12:00 следующего дня.

## Фокус Ревью
- Миграция `pending_inputs` не теряет существующие pending payloads и переносит `thread_id` из payload в явную колонку.
- В коде нет старого глобального ограничения `workspace/user`.
- Worker cleanup не пишет новых сообщений в чат.
- Routine transitions остаются внутри топика `Рутины` и не создают progress events.

## Примечания
- S009 заменяет часть поведения из S008: setup-ввод в другом топике больше не сбрасывает текущий черновик, а просто существует независимо.
- Продакшн пока не трогаем без отдельного approval.
- DB-backed integration tests и `TRACKMATE__DATABASE_URL=... make migrate` не удалось выполнить полноценно: PostgreSQL на `localhost:5432` недоступен, а `make docker-up` не смог подключиться к Docker daemon.

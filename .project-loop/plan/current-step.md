# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-23

## Активный Шаг
- id: `STEP-001`
- status: `готово`
- objective: Локально реализовать два новых топика `Рутины` и `Цели`, аккуратно поправить `Сегодня`, добавить тесты, проверить миграции и подготовить результат на ревью без прод-деплоя.
- requirement IDs: `REQ-001..REQ-014`, `CON-001..CON-004`, `VAL-001..VAL-003`, `SCOPE-001..SCOPE-003`
- owned paths: `.project-loop/`, `internal/`, `cmd/`, `migrations/`, `docs/`, `README.md`, `e2e/telegram/` при необходимости
- validation: `go test ./internal/...`; `make test`; `make lint`; `python3 /Users/igor/plugins/project-loop/skills/project-loop/scripts/loopctl.py validate /Users/igor/projects/trackmate`; DB migration dry-run заблокирован отсутствием PostgreSQL/Docker.
- done criteria: Новые topic flows реализованы и покрыты тестами; existing Today/Progress tests обновлены; миграция additive; routine leaderboard не попадает в `Прогресс`; final response содержит summary, validation, commit hash и план безопасной prod-миграции для approval.

## Фокус Ревью
- Raw intake покрыт требованиями без пропусков.
- Новый код не копирует daily-flow целиком, а выделяет общие механики там, где это реально уменьшает хрупкость.
- Pending inputs и callbacks расширены без поломки existing `daily_task_text`/`daily_task_report`.
- Миграции только добавляют новые данные/индексы/типы или безопасно изменяют enum/contracts; текущая история Today/Progress сохраняется.
- Telegram output остается лаконичным и близким к текущему стилю Trackmate.

## Примечания
- Прод-деплой запрещен до отдельного approval после ревью и миграционного плана.
- PostgreSQL integration/dry-run нужно выполнить перед production: `TRACKMATE_TEST_DATABASE_URL` сейчас не задан, Docker daemon недоступен.

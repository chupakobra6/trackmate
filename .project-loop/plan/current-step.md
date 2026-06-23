# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-23

## Активный Шаг
- id: `STEP-002`
- status: `готово`
- objective: Учесть review delta: fair routine leaderboard, deterministic goal nudge cooldown, вынести новые домены из раздутого service/worker кода, выполнить full Docker/PostgreSQL testing.
- requirement IDs: `REQ-015..REQ-017`, `VAL-004`
- owned paths: `.project-loop/`, `internal/`, `migrations/`, `docs/`, tests
- validation: `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./...`; `make lint`; `make test`; `make migrate`; `python3 /Users/igor/plugins/project-loop/skills/project-loop/scripts/loopctl.py validate /Users/igor/projects/trackmate`
- done criteria: Новая дельта покрыта кодом и тестами; Docker/PostgreSQL validation выполнена; локальный commit создается в конце шага; prod остается без deploy до approval.

## Фокус Ревью
- Leaderboard не sorted/communicated as pure streak-only.
- Goal nudge cooldown хранится в БД и не может спамить чаще 1 раза в 3 дня на пользователя.
- `internal/bot/service.go` и `internal/worker/worker.go` снова остаются маршрутизаторами, а не контейнерами доменной логики.
- PostgreSQL integration tests реально выполняются, не skipped.

## Примечания
- Прод-деплой запрещен до отдельного approval после ревью и миграционного плана.
- Docker теперь доступен; локальные DB tests можно запускать через `TRACKMATE_TEST_DATABASE_URL`.
- Docker compose проверен локально: `postgres`, `api`, `worker` healthy/up; агент может запускать Docker-команды для тестирования.
- Выполнено: `TRACKMATE_TEST_DATABASE_URL=... go test ./...`, `go test ./... -cover`, `make lint`, `TRACKMATE_TEST_DATABASE_URL=... make test`, `TRACKMATE__DATABASE_URL=... make migrate`, `loopctl.py validate`.

# Handoff

Проект: trackmate
Обновлено: 2026-06-23

## Цель
- Реализовать локально новые топики Trackmate: `Рутины` и `Цели`, уточнить `Сегодня`, протестировать, подготовить миграционный план и остановиться перед production approval.

## Текущий Шаг
- active step: `STEP-002`
- status: `готово`

## Завершено
- `.project-loop/` инициализирован.
- Raw-инпут сохранен в `.project-loop/intake/raw/2026-06-23-trackmate-routines-goals.md`.
- Требования, ограничения, валидация и delivery plan нормализованы.
- Локально реализованы `Рутины`, `Цели`, обновление `Сегодня`, additive migration, tests, docs и E2E templates.
- Учтена review delta S003:
  - leaderboard показывает 7-day completion rate, current streak и число пунктов; сортировка идет по completion rate, затем streak;
  - goal nudges работают только при активных целях и имеют DB cooldown 72 часа на пользователя;
  - routine/goals вынесены из `internal/bot/service.go` в `internal/bot/routines.go`, `internal/bot/goals.go`, `internal/app/routine`, `internal/app/goals`;
  - Docker локально доступен, агент может запускать `docker compose` для тестов.

## Измененные Файлы
- `.project-loop/`
- `internal/`, `migrations/`, `docs/`

## Проверка
- `go test ./...`: pass.
- `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./...`: pass.
- `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./... -cover`: pass. Key package coverage: `internal/app/goals` 64.6%, `internal/app/routine` 59.6%, `internal/storage/postgres` 58.6%, `internal/worker` 56.1%, `internal/domain` 63.4%.
- `make lint`: pass.
- `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' make test`: pass.
- `TRACKMATE__DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' make migrate`: pass.
- `loopctl.py validate /Users/igor/projects/trackmate`: pass.

## Агенты
- Subagents отсутствуют.

## Аудит Промптов
- Создается при изменении prompts.

## Пользовательские Дельты
- Отдельный user-deltas stream создается для существенных свежих корректировок, решений или изменений области.

## Риски И Блокеры
- Production migration/deploy заблокированы до approval после локального ревью.
- Перед prod нужен production backup и approval на конкретную command sequence; локальный PostgreSQL dry-run уже выполнен.

## Следующее Действие
- Показать изменения Игорю на ревью. После approval: выполнить production backup/counts, применить migrations, перезапустить сервисы и smoke-check topics.

## Обновленные Источники Правды
- `requirements/source-map.md`
- `requirements/checklist.md`
- `plan/delivery-plan.md`
- `plan/current-step.md`

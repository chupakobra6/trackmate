# Handoff

Проект: trackmate
Обновлено: 2026-06-23

## Цель
- Реализовать локально новые топики Trackmate: `Рутины` и `Цели`, уточнить `Сегодня`, протестировать, подготовить миграционный план и остановиться перед production approval.

## Текущий Шаг
- active step: `STEP-003`
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
- Выполнен live E2E S004 на тестовом боте `@yaminotoubot` в группе `тестирование trackmate v2`.
- Оставлены видимые Telegram-примеры без cleanup/delete:
  - `Сегодня` topic `10`: pinned intro, Today cards, deterministic goal nudge, overdue alert;
  - `Прогресс` topic `11`: закрытые задачи, edited progress event, auto-fail event;
  - `Рутины` topic `339`: pinned intro, routine setup, check-in card, reason/reflection, leaderboard;
  - `Цели` topic `340`: pinned intro, saved goals, weekly review, final period review.
- Найдено и исправлено во время live E2E:
  - time-based E2E waits для уже видимых карточек заменены на `assert_visible_text`;
  - final review целей теперь сравнивает `EndsOn` как локальную календарную дату workspace, а не UTC instant.

## Измененные Файлы
- `.project-loop/`
- `internal/`, `migrations/`, `docs/`, `e2e/telegram/`

## Проверка
- `make docker-up`: pass; `api`, `worker`, `postgres` healthy.
- `go test ./...`: pass.
- `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./...`: pass.
- `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./... -cover`: pass. Key package coverage: `internal/app/goals` 65.3%, `internal/app/routine` 59.6%, `internal/storage/postgres` 58.6%, `internal/worker` 56.1%, `internal/domain` 67.7%.
- `make lint`: pass.
- `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' make test`: pass.
- `TRACKMATE__DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' make migrate`: pass.
- `loopctl.py validate /Users/igor/projects/trackmate`: pass.
- `telegram-bot-e2e-test-tool make doctor`: pass.
- `telegram-bot-e2e-test-tool make test`: pass.
- Live scenarios passed after fixes: `00` setup, `01..11` Today/Progress/alerts, split `12` Routine, split `13` Goals weekly/final, `14` goal nudge.
- Final visible-state evidence: `tmp/e2e-live-logs/98-dump-review-state.log`.

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
- Показать Telegram-примеры Игорю на ревью. После approval миграции: выполнить production backup/counts, применить migrations, перезапустить сервисы и smoke-check topics.

## Обновленные Источники Правды
- `requirements/source-map.md`
- `requirements/checklist.md`
- `plan/delivery-plan.md`
- `plan/current-step.md`

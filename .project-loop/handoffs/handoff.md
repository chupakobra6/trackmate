# Handoff

Проект: trackmate
Обновлено: 2026-06-24

## Цель
- Реализовать локально новые топики Trackmate: `Рутины` и `Цели`, уточнить `Сегодня`, протестировать, подготовить миграционный план и остановиться перед production approval.

## Текущий Шаг
- active step: `STEP-009`
- status: `готово`

## Завершено
- `.project-loop/` инициализирован.
- Raw-инпут сохранен в `.project-loop/intake/raw/2026-06-23-trackmate-routines-goals.md`.
- Требования, ограничения, валидация и delivery plan нормализованы.
- Локально реализованы `Рутины`, `Цели`, обновление `Сегодня`, additive migration, tests, docs и E2E templates.
- Учтена review delta S003:
  - leaderboard показывает 7-day completion rate, current streak и число пунктов; сортировка идет по completion rate, затем streak;
  - вставки про цели работают только при активных целях и имеют ограничение 72 часа на пользователя;
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
- Закрыт STEP-004 по текстам:
  - пользовательские сообщения Today/Routine/Goals/Progress/alerts стали лаконичнее и ровнее по отступам;
  - видимые термины заменены: check-in -> проверка, leaderboard -> таблица, review -> обзор/итог периода, streak/стрик -> серия, report -> итог;
  - кнопка ежедневной задачи стала `🏁 Подвести итог`;
  - E2E-шаблоны обновлены под новые видимые тексты.
- Закрыт STEP-005 по внешнему ревью текстов S006:
  - `цель-задача дня` заменено на `задача дня` в пользовательских текстах;
  - карточки `Сегодня`/`Прогресс` стали компактнее: `План:` и `Итог:` идут сразу перед цитатой;
  - вставки про цели смягчены без слов `провал` и `двигает тебя`;
  - `Рутины: таблица` заменено на `Таблица рутин`;
  - шаблон целей структурирован маркерами, при этом конфликтующий пример `оффер Go/backend` не возвращен.
- Закрыт STEP-006 по финальному ревью текстов S007:
  - закрепы `Сегодня`, `Рутины`, `Цели`, `Прогресс` приведены к спокойному стилю старых топиков;
  - видимые списки переведены с `•` на длинное тире `—`;
  - `Цели` описаны как долгосрочные цели на сезон, а prompt объясняет поля формата без частного примера;
  - недельный обзор целей и итог периода переписаны на конкретные вопросы;
  - parser рутины принимает `—` в пользовательском списке.
- Production `v1.1` выкачен на VPS `inferno-nl` в `/opt/trackmate`: commit/tag `c97c222`/`v1.1`, backup `/opt/trackmate/backups/trackmate_20260623T023521Z.dump`, миграции до `202606230003`, `api`/`worker`/`postgres` healthy, анонс `Trackmate 1.1` опубликован в `Прогресс` message `3280`.
- Закрыт STEP-009 по UX после production:
  - незаконченные setup-черновики `routine_plan`/`seasonal_goals` теперь сбрасываются при переходе в другой setup-топик;
  - старый prompt Trackmate и wrong-topic сообщение пользователя удаляются для таких черновиков;
  - цели сохраняются с единым коротким ответом без отдельной карточки с полным текстом целей;
  - старые goal card messages удаляются при следующем сохранении целей;
  - в текстах рутины явно указано, что ежедневная карточка приходит после 09:00.

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
- Live scenarios passed after fixes: `00` setup, `01..11` Today/Progress/alerts, split `12` Routine, split `13` Goals weekly/final, `14` вставка про цели.
- Final visible-state evidence: `tmp/e2e-live-logs/98-dump-review-state.log`.
- STEP-004 правка текстов: `go test ./internal/ui ./internal/bot ./internal/app/goals ./internal/app/routine`: pass.
- STEP-004 правка текстов: `make test`: pass.
- STEP-004 правка текстов: `make lint`: pass.
- STEP-005 внешнее ревью текстов: `go test ./internal/ui ./internal/bot ./internal/app/goals ./internal/app/routine ./internal/storage/postgres`: pass.
- STEP-005 внешнее ревью текстов: `make test`: pass.
- STEP-005 внешнее ревью текстов: `make lint`: pass.
- STEP-006 финальное ревью текстов: `go test ./internal/ui ./internal/domain ./internal/bot ./internal/app/goals ./internal/app/routine`: pass.
- STEP-006 финальное ревью текстов: `make test`: pass.
- STEP-006 финальное ревью текстов: `make lint`: pass.
- STEP-006 финальное ревью текстов: `loopctl.py validate /Users/igor/projects/trackmate`: pass.
- STEP-009 UX после production: `go test ./internal/bot ./internal/ui ./internal/storage/postgres ./internal/domain ./internal/app/routine ./internal/app/goals`: pass.
- STEP-009 UX после production: `make test`: pass.
- STEP-009 UX после production: `make lint`: pass.
- STEP-009 UX после production: `loopctl.py validate /Users/igor/projects/trackmate`: pass.

## Агенты
- Subagents отсутствуют.

## Аудит Промптов
- Создается при изменении prompts.

## Пользовательские Дельты
- Отдельный user-deltas stream создается для существенных свежих корректировок, решений или изменений области.

## Риски И Блокеры
- Текущий STEP-009 закоммичен локально, но не выкачен на production.
- Для production hotfix нужен отдельный approval, затем короткий update без новых миграций.

## Следующее Действие
- Показать STEP-009 Игорю. После approval на hotfix: push, production backup/counts, `git pull --ff-only`, `docker compose up -d --build`, smoke-check pending cleanup и goals confirmation.

## Обновленные Источники Правды
- `requirements/source-map.md`
- `requirements/checklist.md`
- `plan/delivery-plan.md`
- `plan/current-step.md`

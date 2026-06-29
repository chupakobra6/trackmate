# Handoff

Проект: trackmate
Обновлено: 2026-06-29

## Цель
- Реализовать локально новые топики Trackmate: `Рутины` и `Цели`, уточнить `Сегодня`, протестировать, подготовить миграционный план и остановиться перед production approval.

## Текущий Шаг
- active step: `STEP-012`
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
- Закрыт STEP-010 по S009:
  - `pending_inputs` теперь topic-scoped: явный `message_thread_id`, уникальность `workspace/user/thread`, сообщения и callbacks ищут pending только в текущей теме;
  - поведение S008 со сбросом setup-черновиков между `Рутины` и `Цели` заменено: чужой топик не сбрасывается, wrong-topic сообщения игнорируются;
  - worker тихо чистит pending старше 24 часов и удаляет сохраненные prompt/user message IDs без сообщений в чат;
  - рутина приходит после 20:00 локального времени; если список сохранен до 20:00, первая карточка может прийти в тот же день;
  - для незакрытой рутины worker отправляет напоминание после конца дня и в 12:00 следующего дня закрывает неотмеченные пункты как `failed`;
  - автозакрытие рутины остается в `Рутины`, не создает `progress_events`, обновляет карточку и таблицу рутин.
- Закрыт STEP-011 по S010:
  - объяснение production-семантики: текущая дата в карточке рутины означает дату проверки рутины, поэтому `24.06` — это рутина за 24 июня, не за 23 июня;
  - карточка рутины теперь пишет `Рутина за DD.MM` и сразу под заголовком `Отметь, как прошел этот день`;
  - prompt причины получил такой же заголовок и пояснение, чтобы дата не терялась при ответах `Нет`/`Частично`;
  - production не трогался в этом шаге.
- После approval STEP-009..STEP-011 выкачены на production: VPS `/opt/trackmate` сейчас на `9a58215`, миграция `202606240001` применена, `api`/`worker`/`postgres` healthy.
- Закрыт STEP-012 по S011:
  - production проверен read-only: logs 2026-06-24 17:00 UTC показали routine cards `3373`/`3374`, затем callbacks по `3373` и edit в reason prompt;
  - причина разного вида со скриншота: `FormatRoutineReasonPrompt` строил текст отдельно, без автора и без обычного формата `N/M: пункт?`;
  - reason prompt теперь переиспользует обычную карточку рутины, сохраняет автора и формат вопроса, добавляя только `Что помешало?`;
  - первый emoji routine header/control заменен с `🔁` на `🌿`;
  - production deploy не выполнялся в STEP-012.

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
- STEP-010 topic-scoped pending/routine flow: `go test ./internal/domain ./internal/storage/postgres ./internal/app/pending ./internal/app/routine ./internal/app/goals ./internal/worker ./internal/bot ./internal/ui`: pass.
- STEP-010: `go test ./... -count=1`: pass.
- STEP-010: `make test`: pass.
- STEP-010: `make lint`: pass.
- STEP-010 DB-backed integration tests and migration dry-run: blocked locally because PostgreSQL was not listening on `localhost:5432` and `make docker-up` could not connect to Docker daemon.
- STEP-011 routine date copy: `go test ./internal/ui ./internal/bot ./internal/app/routine`: pass.
- STEP-011: `make lint`: pass.
- STEP-011: `loopctl.py validate /Users/igor/projects/trackmate`: pass.
- STEP-011: `git diff --check`: pass.
- STEP-012 production read-only check: VPS `/opt/trackmate` at `9a58215`, docker compose healthy; DB/logs around 2026-06-24 20:00 MSK inspected.
- STEP-012: `go test ./internal/ui ./internal/bot`: pass.
- STEP-012: `go test ./... -count=1`: pass.
- STEP-012: `make lint`: pass.
- STEP-012: `loopctl.py validate /Users/igor/projects/trackmate`: pass.

## Агенты
- Subagents отсутствуют.

## Аудит Промптов
- Создается при изменении prompts.

## Пользовательские Дельты
- Отдельный user-deltas stream создается для существенных свежих корректировок, решений или изменений области.

## Риски И Блокеры
- STEP-012 локально готов, но не выкачен на production по прямой инструкции Игоря; включить в будущую пачку исправлений.
- В STEP-012 production данные не чистились и не менялись.

## Следующее Действие
- Ждать следующий скриншот/дельту. Текущий локальный fix можно будет выкатить позже вместе с пачкой исправлений после отдельной команды.

## Обновленные Источники Правды
- `requirements/source-map.md`
- `requirements/checklist.md`
- `plan/delivery-plan.md`
- `plan/current-step.md`

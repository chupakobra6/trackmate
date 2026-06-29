# Handoff

Проект: trackmate
Обновлено: 2026-06-29

## Цель
- Реализовать локально новые топики Trackmate: `Рутины` и `Цели`, уточнить `Сегодня`, протестировать, подготовить миграционный план и остановиться перед production approval.

## Текущий Шаг
- active step: `STEP-019`
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
- Закрыт STEP-013 по S012:
  - production проверен read-only: текущий prod на `9a58215`, logs 2026-06-24 17:00 UTC подтверждают, что старый flow редактировал основную карточку для причины;
  - `Нет`/`Частично` теперь сразу отмечают пункт в основной карточке и отправляют отдельный временный reason prompt reply к карточке;
  - после ответа пользователя Trackmate удаляет reason prompt и ответ пользователя, сохраняет причину и двигает основную карточку дальше;
  - после последнего пункта routine check-in закрывается сразу, обновляет карточку и таблицу рутин, без routine final reflection;
  - active `routine_reflection` path удален из bot/domain/storage API; историческое поле `reflection_text` в схеме не трогалось;
  - цели не менялись, production deploy не выполнялся.
- Закрыт STEP-014 по S013:
  - production incident по Егору восстановлен без новых сообщений в чат: `daily_tasks.id=160` теперь `done`, `report_status=done`, `report_text=Голосовое сообщение`, `report_message_id=3386`, `failed_at=NULL`;
  - неверный auto-fail `progress_events.id=183` удален из БД, новую публикацию в `Прогресс` не создавали;
  - карточка задачи `3361` в `Сегодня` отредактирована в фактическое состояние `выполнена`;
  - production backup перед правкой: `/opt/trackmate/backups/trackmate_manual_fix_20260629T093903Z.dump`;
  - root cause подтвержден логами: `task:status:160:done` 2026-06-24 19:33 UTC упал на `duplicate key value violates unique constraint "uq_pending_inputs_workspace_group_id"`;
  - production schema исправлена вручную: старый constraint `uq_pending_inputs_workspace_group_id` удален, остался `ux_pending_inputs_workspace_user_thread`;
  - старые Telegram messages `3385` и `3404` удалить не удалось: Bot API вернул `Bad Request: message can't be deleted`;
  - локально добавлена миграция `202606290001_drop_legacy_pending_input_unique.sql`, production deploy кода не выполнялся.
- Закрыт STEP-015 по S014:
  - все пользовательские тексты, подписи кнопок, callback-ответы, media labels и названия сезонов вынесены в `internal/messages/messages.md`;
  - добавлен `internal/messages` с `go:embed`-импортом и тестами загрузки/шаблонов;
  - production Go-код больше не содержит русских user-facing string literals вне каталога сообщений;
  - prompt рутины упрощен: пример только через дефисы `-`, без текста про максимум;
  - routine parser теперь принимает только пункты с `-` или `—`, номера/буллеты/свободные строки отклоняет;
  - основная карточка рутины больше не показывает `1/N: пункт?`, а только дату, пояснение и список статусов;
  - служебные notice/problem messages получили общий `notice:dismiss` и кнопку `👀 Понял`;
  - routine reminders/auto-close notices отправляются с `Понял`, а старое напоминание удаляется при автозакрытии/ручном закрытии;
  - при успешном сохранении рутины/целей удаляются сохраненные ошибочные user messages и текущий setup input, prompt редактируется в короткое подтверждение с `Понял`;
  - production deploy не выполнялся.
- Закрыт STEP-016 по S015:
  - routine setup больше не оставляет confirmation-сообщение: после успешного ввода удаляются prompt Trackmate, пользовательский список и сохраненные ошибочные user messages;
  - routine parser снова принимает нумерацию `1.`/`2)` как префикс пункта, порядок номеров не проверяется;
  - prompt рутины остается с примером на дефисах и коротко пишет, что нумерацию тоже поймет;
  - после полного ответа на daily routine карточка удаляется, таблица рутин обновляется;
  - auto-close рутины удаляет карточку, напоминание и pending reason prompt/user messages без отдельного `Время вышло` notice;
  - production deploy не выполнялся.
- Закрыт STEP-017 по S016:
  - routine reminder переписан в короткий стиль: `Рутина за DD.MM`, `Закрой до 12:00`, `Неотмеченные пункты станут невыполненными`;
  - reminder использует кнопку `👀 Понял`, удаляется при ручном закрытии рутины и чистится worker cleanup после TTL около суток;
  - auto-close notice возвращен как временный alert: `Рутина за DD.MM закрыта`, `Неотмеченные пункты стали невыполненными`;
  - добавлена additive migration `202606290002_routine_notice_ttl.sql` с nullable-полями `auto_close_notice_message_id` и `auto_close_notice_sent_at`;
  - worker теперь чистит expired routine reminder и auto-close notice messages после `RunCheckinTransitions`;
  - production deploy не выполнялся.
- Закрыт STEP-018 по S017:
  - production-БД проверена по task `160`/message `3386`: task уже был `done`, `report_status=done`, `report_text=Голосовое сообщение`, `failed_at=NULL`;
  - missing piece: для `daily_task_id=160` не было `progress_events.daily_task.closed`, поэтому история `Прогресс` была неполной;
  - перед правкой снят backup `/opt/trackmate/backups/trackmate_manual_progress_fix_20260629T111509Z.dump`;
  - попытка отредактировать старое message `3404` в корректный done-progress не прошла: Bot API вернул `Bad Request: message to edit not found`;
  - создан `progress_events.id=192` с `event_type=daily_task.closed`, `status=done`, `report_html=Голосовое сообщение`, `created_at=2026-06-24 19:35:27.83718+00`;
  - worker штатно опубликовал event в `Прогресс`: Telegram message `3649`, `publish_status=published`;
  - локальный код и production deploy не трогались.
- Закрыт STEP-019 по S018:
  - production logs 2026-06-26 08:49-08:55 MSK подтвердили root cause жалобы Егора: `today:add` падал на legacy constraint `uq_pending_inputs_workspace_group_id`;
  - production schema сейчас исправлена: legacy constraint отсутствует, rollback probe успешно создает два pending inputs одного пользователя в разных топиках;
  - сообщения жалобы в общем чате `3464` и `3465` удалены через Telegram Harvest main profile, повторный dump их не находит;
  - задача Егора за 2026-06-26 уже восстановлена как `daily_tasks.id=172`, status `partial`, task message `3467`;
  - voice `3386` расшифрован через Harvest/Vosk; сырой ASR шумный, поэтому сохранена вычитанная смысловая сводка;
  - перед правкой снят backup `/opt/trackmate/backups/trackmate_manual_voice_transcript_20260629T121722Z.dump`;
  - `daily_tasks.id=160`, `progress_events.id=192`, visible Today message `3361` и Progress message `3649` обновлены без новых Telegram posts;
  - локальный код и production deploy не трогались.

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
- STEP-013 production read-only check: VPS `/opt/trackmate` at `9a58215`, docker compose healthy; logs around 2026-06-24 20:00 MSK inspected.
- STEP-013: `go test ./internal/ui ./internal/bot ./internal/app/routine ./internal/storage/postgres`: pass.
- STEP-013: `go test ./... -count=1`: pass.
- STEP-013: `make lint`: pass.
- STEP-013: `loopctl.py validate /Users/igor/projects/trackmate`: pass.
- STEP-014 production verification: task `160` fixed; legacy pending constraint removed; `api`/`worker`/`postgres` healthy.
- STEP-014: `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./internal/storage/postgres ./internal/bot ./internal/app/pending`: pass.
- STEP-014: `git diff --check`: pass.
- STEP-014: `make test`: pass.
- STEP-014: `make lint`: pass.
- STEP-014: `TRACKMATE__DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' make migrate`: pass.
- STEP-014: `loopctl.py validate /Users/igor/projects/trackmate`: pass.
- STEP-015 message catalog/routine input: `go test ./internal/messages ./internal/domain ./internal/ui ./internal/telegram ./internal/app/goals ./internal/app/routine ./internal/bot ./internal/worker`: pass.
- STEP-015: `make test`: pass.
- STEP-015: `make lint`: pass.
- STEP-015: `git diff --check`: pass.
- STEP-015: `loopctl.py validate .`: pass.
- STEP-016 routine cleanup/numbering: `go test ./internal/domain ./internal/ui ./internal/bot ./internal/app/routine ./internal/messages`: pass.
- STEP-016: `make test`: pass.
- STEP-016: `make lint`: pass.
- STEP-016: `git diff --check`: pass.
- STEP-016: `loopctl.py validate .`: pass.
- STEP-017 routine alerts TTL/copy: `go test ./internal/storage/postgres ./internal/app/routine ./internal/ui ./internal/domain`: pass.
- STEP-017: `go test ./internal/bot ./internal/worker ./internal/messages`: pass.
- STEP-017: `make test`: pass.
- STEP-017: `make lint`: pass.
- STEP-017: `git diff --check`: pass.
- STEP-017: `loopctl.py validate .`: pass.
- STEP-018 production SQL verification: task `160` done/report `3386`, alerts acknowledged with no message ids, progress event `192` published as message `3649`.
- STEP-018 worker log verification: `telegram_send_message_completed` for message `3649` in thread `7`.
- STEP-018: `loopctl.py validate .`: pass.
- STEP-019 production log verification: old pending unique constraint was root cause for message `3467` not saving on 2026-06-26.
- STEP-019 production schema verification: rollback probe accepts same user pending inputs in two different topics.
- STEP-019 Harvest cleanup verification: messages `3464`/`3465` deleted and absent from follow-up dump.
- STEP-019 DB/message verification: task `160`, task `172`, progress `192`, messages `3361`/`3649` checked after edits.

## Агенты
- Subagents отсутствуют.

## Аудит Промптов
- Создается при изменении prompts.

## Пользовательские Дельты
- Отдельный user-deltas stream создается для существенных свежих корректировок, решений или изменений области.

## Риски И Блокеры
- STEP-012, STEP-013, STEP-015, STEP-016 и STEP-017 локально готовы, но не выкачены на production по прямой инструкции Игоря; включить в будущую пачку исправлений.
- STEP-014 production data/schema исправлены вручную; локальная миграция добавлена, но кодовый deploy не выполнялся.
- STEP-018 production data исправлен вручную; кодовый deploy не выполнялся.
- STEP-019 production data/message cleanup выполнен вручную; кодовый deploy не выполнялся.

## Следующее Действие
- Ждать следующий скриншот/дельту или отдельную команду на deploy. Текущие локальные fixes STEP-012/STEP-013/STEP-015/STEP-016/STEP-017 и миграции STEP-014/STEP-017 можно будет выкатить позже вместе с пачкой исправлений после отдельной команды.

## Обновленные Источники Правды
- `requirements/source-map.md`
- `requirements/checklist.md`
- `plan/delivery-plan.md`
- `plan/current-step.md`

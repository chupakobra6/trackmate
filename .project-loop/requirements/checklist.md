# Чеклист Требований

Проект: trackmate
Обновлено: 2026-06-23

## Значения Статусов
Используй `кандидат`, `принято`, `в работе`, `готово`, `отложено`, `заблокировано` или `отклонено`.

## Требования
| ID | Статус | Источник | Требование | Критерий приемки | Доказательства |
| --- | --- | --- | --- | --- | --- |
| REQ-001 | `готово` | S001 | Сохранить raw-инпут для сверки требований. | Исходный текст лежит в `.project-loop/intake/raw/2026-06-23-trackmate-routines-goals.md` и зарегистрирован в source map. | `.project-loop/intake/raw/2026-06-23-trackmate-routines-goals.md` |
| REQ-002 | `готово` | S001 | Обновить топик `Сегодня`: оставить текущую механику, но в лаконичной инструкции явно просить одну главную цель-задачу дня; формат ввода остается свободным. | Закрепленная карточка/текст Today говорит про одну главную цель-задачу дня, не ломает текущий daily-flow. | `internal/ui/formatters.go`, `make test` |
| REQ-003 | `готово` | S001 | Создать топик `Рутины` для повторяемых ежедневных действий. | Setup создает/поддерживает topic binding, закрепленную карточку с одной кнопкой `✏️ Настроить рутину`, отдельные тексты и callback flow. | `internal/app/setup/setup.go`, `internal/ui/keyboards.go`, `make test` |
| REQ-004 | `готово` | S001 | Реализовать настройку рутины списком. | Бот принимает непустые строки, чистит маркеры `-`, `•`, `1.`, пробелы; максимум 9 пунктов; на MVP все пункты ежедневные. | `internal/domain/rules.go`, `internal/domain/rules_test.go` |
| REQ-005 | `готово` | S001 | Реализовать ежедневный routine check-in без спама. | Для пользователя публикуется/редактируется одна карточка в `Рутины`; пункты проходятся item-by-item кнопками `Да`/`Частично`/`Нет`; причины для `Частично`/`Нет`; финальная рефлексия: `Что помогло / что помешало / какую одну правку сделаешь завтра?`. | `internal/bot/routines.go`, `internal/app/routine/routine.go`, integration tests added |
| REQ-006 | `готово` | S001 | Сохранять routine check-in, причины и итоговую рефлексию, но не публиковать их в `Прогресс`. | Данные есть в БД для истории/статистики; `Прогресс` не получает routine events; итоговая карточка пользователя остается в `Рутины`. | `internal/storage/postgres/routines.go`, bot integration test added |
| REQ-007 | `готово` | S001 | Реализовать routine streaks и leaderboard в `Рутины`. | `done=1`, `partial=0.5`, `failed=0`; полный день = все пункты done; current streak, max streak, 7-day completion rate; leaderboard публикуется только в topic `Рутины`. | `internal/storage/postgres/routines.go`, `internal/ui/formatters.go` |
| REQ-008 | `готово` | S001 | Создать топик `Цели` для сезонных целей. | Setup создает/поддерживает topic binding, закрепленную карточку, кнопку настройки целей и инструкции. Первый период: `Лето 2026` до `2026-09-01`. | `internal/app/setup/setup.go`, `internal/ui/formatters.go` |
| REQ-009 | `готово` | S001 | Формат целей должен направлять к измеримым целям. | Инструкция просит гибрид SMART/OKR: `Результат`, `Метрика`, `Еженедельный шаг`, `Почему важно`; raw goals сохраняются и показываются в карточке. | `internal/ui/formatters.go`, `internal/storage/postgres/goals.go` |
| REQ-010 | `готово` | S001 | Реализовать weekly review целей. | Раз в неделю в конце недели бот просит общий текст: что сдвинулось, что мешало, главный шаг на следующую неделю; без опроса по каждому пункту; ответ сохраняется. | `internal/app/goals/goals.go`, `internal/bot/goals.go` |
| REQ-011 | `готово` | S001 | Реализовать финальный review периода. | На дату окончания периода бот просит оценку целей `Выполнены`/`Частично`/`Не выполнены`, затем короткий итог: что получилось, что нет, вывод на следующий сезон. | `internal/app/goals/goals.go`, `internal/bot/goals.go` |
| REQ-012 | `готово` | S001 | Добавить редкие связующие напоминания между `Сегодня` и сезонными целями. | При постановке задачи на день и при закрытии/частичном/провале с небольшим шансом показывается короткий prompt про связь с целями. | `internal/app/goals/goals.go`, `internal/domain/rules.go`, `internal/bot/goals.go` |
| REQ-013 | `готово` | S001 | Сохранить данные и подготовить безопасную прод-миграцию. | Новая схема добавляется миграциями без destructive операций над текущими history/stat tables; миграционный план описан до прод-деплоя; деплой только после approval. | `migrations/202606230001_add_routines_and_goals.sql`, `.project-loop/plan/prod-migration-plan.md` |
| REQ-014 | `готово` | S001,S002 | Не разрастить код копипастой. | Общие механики pending input/callback/topic/card/worker переиспользуются или аккуратно обобщаются, без параллельных legacy contracts. | `pending_inputs`, callback parser, topic setup, worker dispatchers reused |
| REQ-015 | `готово` | S003 | Сделать routine leaderboard честнее для разного числа пунктов. | Leaderboard явно показывает current streak и 7-day completion rate; ranking не выглядит чистым streak-only; item count виден или понятен. | `internal/storage/postgres/routines.go`, `internal/ui/formatters.go`, `TestRoutineLeaderboardRanksCompletionRateBeforeStreak` |
| REQ-016 | `готово` | S003 | Заменить чистый random goal nudge на deterministic throttle. | Goal nudges показываются только при активных целях, не чаще 1 раза в 3 дня на пользователя; внутри остается deterministic pseudo-random. | `migrations/202606230002_add_goal_nudge_cooldowns.sql`, `internal/app/goals/goals.go`, `TestMaybeNudgeUsesActiveGoalsAndThreeDayCooldown` |
| REQ-017 | `готово` | S003 | Уменьшить раздувание `internal/bot/service.go` и отделить новые домены. | Routine/goals Telegram handlers вынесены из основного service file; worker orchestration использует отдельные app packages для routine/goals. | `internal/bot/routines.go`, `internal/bot/goals.go`, `internal/app/routine/routine.go`, `internal/app/goals/goals.go`, `internal/worker/worker.go` |
| REQ-018 | `готово` | S004 | Оставить после live E2E видимые примеры workflow в тестовой Telegram-группе. | В темах `Сегодня`, `Рутины`, `Цели`, `Прогресс` остаются сообщения/карточки, по которым можно оценить текст и вид; cleanup/delete сценарии не запускались. | Тестовая группа `тестирование trackmate v2`; topics: `Сегодня` 10, `Прогресс` 11, `Рутины` 339, `Цели` 340; `tmp/e2e-live-logs/98-dump-review-state.log`. |

## Ограничения
| ID | Статус | Источник | Ограничение | Доказательства |
| --- | --- | --- | --- | --- |
| CON-001 | `принято` | S001 | Минимизировать спам: routine check-in редактирует одну карточку пользователя, а не шлет отдельное сообщение на каждый пункт. |  |
| CON-002 | `принято` | S001 | Не публиковать рутины, причины и leaderboard в `Прогресс`. |  |
| CON-003 | `принято` | S001 | Не терять текущие данные БД для истории и статистики. |  |
| CON-004 | `принято` | S001 | На прод не выкатывать без отдельного approval после ревью локальной реализации и миграционного плана. |  |
| CON-005 | `принято` | S004 | Не удалять E2E сообщения после текущего live-прогона. |  |

## Обязательная Валидация
| ID | Статус | Источник | Валидация | Доказательства |
| --- | --- | --- | --- | --- |
| VAL-001 | `готово` | S001,S002,S003 | Добавить/обновить Go unit/integration tests для новых парсеров, callbacks, formatters, storage и worker flows по мере затронутого кода. | Domain/UI unit tests; bot/storage/worker/app PostgreSQL integration tests. |
| VAL-002 | `готово` | S001,S002 | Запустить focused tests по измененным пакетам, затем `make test` и `make lint`, если feasible. | `go test ./internal/...`; `make lint`; `make test` |
| VAL-003 | `готово` | S001,S002 | Для миграций проверить применение на локальной БД/тестовой схеме, не destructive для существующих таблиц. | `TRACKMATE_TEST_DATABASE_URL=... go test ./...`; `TRACKMATE__DATABASE_URL=... make migrate`; миграции additive для текущей истории. |
| VAL-004 | `готово` | S003 | После Docker availability выполнить полное тестирование с PostgreSQL. | `docker compose ps`; `TRACKMATE_TEST_DATABASE_URL=... go test ./...`; `go test ./... -cover`; `make lint`; `make test`; `make migrate`; `loopctl.py validate`. |
| VAL-005 | `готово` | S004 | Выполнить полный live E2E на тестовом Telegram-боте по новым workflow и исправить найденные ошибки. | Setup/Today/Routine/Goals/Progress/alerts scenarios через `telegram-bot-e2e-test-tool`; failing scenarios повторены после fixes; final review timezone regression covered by `TestGoalFinalReviewDueUsesWorkspaceLocalDate`. |

## Границы Объема
| ID | Статус | Источник | Граница | Примечания |
| --- | --- | --- | --- | --- |
| SCOPE-001 | `принято` | S001 | В текущем проходе делаем локальную реализацию, тесты и показываем на ревью. | Прод-деплой и реальные миграции на VPS только после отдельного approval. |
| SCOPE-002 | `принято` | S001 | MVP routine items все ежедневные. | Расписание по разным дням не входит в первый шаг. |
| SCOPE-003 | `принято` | S001 | Goals MVP сохраняет raw text целей. | Идеальный парсинг каждого поля не обязателен, но инструкция должна вести к структуре. |

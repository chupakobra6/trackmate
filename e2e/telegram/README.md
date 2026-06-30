# Telegram E2E

Здесь лежат только продуктовые сценарии Trackmate. Общий запускатель, MTProto-сессия,
локи, бинарные файлы для проверок и записи прогонов остаются в соседнем репозитории
`/Users/igor/projects/telegram-bot-e2e-test-tool`.

## Что Проверяем

- подготовка создает и чинит темы `Сегодня`, `Рутины`, `Цели` и `Прогресс`;
- задача дня создается только в теме `Сегодня`;
- вторая открытая задача для того же участника блокируется;
- итоги закрывают задачу со состояниями `выполнена`, `выполнена частично`,
  `не выполнена`;
- ожидаемый ввод не съедает сообщение из другой темы;
- фотоальбом в итоге закрывает задачу один раз;
- рабочий процесс публикует событие в теме `Прогресс`;
- редактирование исходного сообщения итога обновляет `Сегодня` и `Прогресс`;
- если Telegram не смог отредактировать старое сообщение после сохраненного
  изменения, Go integration test проверяет системный алерт в `Прогресс`;
- напоминание подтверждается кнопкой `👀 Понял`;
- ежедневная проверка рутины идет в теме `Рутины` и не публикуется в `Прогресс`;
- вопросы по целям раз в две недели и итог периода идут в теме `Цели`;
- редкая вставка про цели на сезон появляется в `Сегодня` только при активных целях.

## Подготовка

Запускай сценарии только против локальной или явно тестовой forum supergroup.
Продакшн-бот и продакшн-БД не использовать.

```bash
cd /Users/igor/projects/trackmate
make docker-up
```

В соседнем runner-репозитории:

```bash
cd /Users/igor/projects/telegram-bot-e2e-test-tool
make fixtures
make doctor
go test ./...
```

## Темы

Сначала запусти `scenarios/00-setup-smoke.jsonl`, чтобы бот создал или починил
темы. Потом получи MTProto target и topic ids:

```bash
cd /Users/igor/projects/telegram-bot-e2e-test-tool
make chats CHAT_GROUPS=1 CHAT_FILTER=TrackMate CHAT_TOPICS=1 CHAT_ADMINS=1
```

Нужны значения:

- `target=` -> `TRACKMATE_CHAT`;
- `topic_id=` для `Сегодня` -> `TODAY_THREAD_ID`;
- `topic_id=` для `Рутины` -> `ROUTINE_THREAD_ID`;
- `topic_id=` для `Цели` -> `GOALS_THREAD_ID`;
- `topic_id=` для `Прогресс` -> `PROGRESS_THREAD_ID`;
- `WRONG_THREAD_ID` можно поставить равным `PROGRESS_THREAD_ID` для проверки
  wrong-topic pending input. Черновики в разных темах изолированы: сообщение в
  чужой теме не закрывает и не сбрасывает текущий ввод.

## Рендер Темплейтов

```bash
export TRACKMATE_CHAT='3871708263'
export TODAY_THREAD_ID='10'
export ROUTINE_THREAD_ID='11'
export GOALS_THREAD_ID='12'
export PROGRESS_THREAD_ID='13'
export WRONG_THREAD_ID="$PROGRESS_THREAD_ID"

mkdir -p tmp/e2e-rendered
for src in e2e/telegram/scenarios/*.jsonl.tmpl; do
  dst="tmp/e2e-rendered/$(basename "${src%.tmpl}")"
  sed \
    -e "s/{{CHAT}}/$TRACKMATE_CHAT/g" \
    -e "s/{{TODAY_THREAD_ID}}/$TODAY_THREAD_ID/g" \
    -e "s/{{ROUTINE_THREAD_ID}}/$ROUTINE_THREAD_ID/g" \
    -e "s/{{GOALS_THREAD_ID}}/$GOALS_THREAD_ID/g" \
    -e "s/{{PROGRESS_THREAD_ID}}/$PROGRESS_THREAD_ID/g" \
    -e "s/{{WRONG_THREAD_ID}}/$WRONG_THREAD_ID/g" \
    "$src" > "$dst"
done
```

Запуск одного сценария:

```bash
cd /Users/igor/projects/telegram-bot-e2e-test-tool
CHAT="$TRACKMATE_CHAT" go run ./cmd/tg-e2e-tool run-scenario \
  /Users/igor/projects/trackmate/tmp/e2e-rendered/02-today-create-task.jsonl
```

`00-setup-smoke.jsonl` не темплейтится и запускается напрямую с `CHAT`.
Сценарии не содержат `dump_state`: запускатель сам пишет записи прогона, а
продуктовые файлы здесь описывают только проверяемый пользовательский путь.
Кнопки на карточках задач кликаются с `message_text`, чтобы в давно используемой
тестовой группе не попасть в старую кнопку из истории.
Если сценарий проверяет сообщение, которое могло появиться до шага ожидания,
используй `assert_visible_text`, а не `wait`: `wait` ждет новое изменение истории
и может флапать на уже видимом корректном состоянии.
Для карточек и важных сообщений проверяй итоговый вид целиком или максимально
близко к полному тексту, который видит пользователь. Проверка одного короткого
фрагмента допустима только как дополнительная страховка, а не как основное
доказательство правильного текста.

Сообщения, которые пользователь должен видеть как отдельные подсказки в теме,
должны отправляться обычными корневыми сообщениями. Вложенный ответ к карточке
бота может не попасть в историю темы, из-за чего живой пользователь видит одно,
а MTProto-снимок темы для E2E — другое.

После полного прогона можно очистить видимые тестовые сообщения в темах:

```bash
cd /Users/igor/projects/telegram-bot-e2e-test-tool
CHAT="$TRACKMATE_CHAT" go run ./cmd/tg-e2e-tool run-scenario \
  /Users/igor/projects/trackmate/tmp/e2e-rendered/99-cleanup-visible-messages.jsonl
```

Очистка не удаляет темы, служебные сообщения Telegram, стартовое сообщение темы и закрепленное
сообщение. Для удаления сообщений бота в супергруппе MTProto-пользователь запускателя
должен иметь права на удаление сообщений.

Для ревью-прогонов, где нужно оставить визуальные примеры в Telegram, очистка не
запускается. В таком режиме достаточно в конце сделать `dump_state` по темам или
сверить сообщения прямо в тестовой группе.

Если сценарий создает временные алерты, напоминания или ошибки с кнопками
подтверждения, сценарий должен нажать `👀 Понял` или соответствующую кнопку
завершения и проверить, что сообщение исчезло или перешло в ожидаемое конечное
состояние. После обычного E2E-прогона запускай cleanup-сценарий, чтобы старые
сообщения не смешивались с новой ручной проверкой.

## Сценарии

- `00-setup-smoke.jsonl`: `/setup`, повторная проверка условий, оформление группы.
- `01-today-add-prompt.jsonl.tmpl`: кнопка добавления задачи.
- `02-today-create-task.jsonl.tmpl`: создание задачи дня.
- `03-today-block-second-task.jsonl.tmpl`: запрет второй задачи.
- `04-report-done.jsonl.tmpl`: итог `✅ Выполнена`.
- `05-report-partial.jsonl.tmpl`: итог `🔸 Частично`.
- `06-report-failed.jsonl.tmpl`: итог `❌ Не выполнена`.
- `07-wrong-topic-pending-ignored.jsonl.tmpl`: daily pending input не закрывается
  сообщением из другой темы.
- `08-duplicate-photo-report-consumes-once.jsonl.tmpl`: фотоальбом Telegram
  из двух фото с общей подписью закрывает итог один раз.
- `09-progress-topic-event.jsonl.tmpl`: событие появляется в `Прогресс`.
- `10-alert-ack.jsonl.tmpl`: кнопка `👀 Понял` подтверждает напоминание.
- `11-edited-report-progress-sync.jsonl.tmpl`: правка исходного сообщения итога
  обновляет карточку в `Сегодня` и опубликованное событие в `Прогресс`.
- `12-routine-checkin.jsonl.tmpl`: настройка рутины и утренняя проверка за предыдущий день в `Рутины`.
- `13-goals-weekly-final.jsonl.tmpl`: настройка целей на сезон, вопросы по целям и
  итог периода в `Цели`.
- `14-goal-nudge.jsonl.tmpl`: редкая детерминированная вставка про цели на сезон при
  постановке задачи дня, если у участника есть активные цели.
- `99-cleanup-visible-messages.jsonl.tmpl`: удаляет видимые тестовые сообщения в
  `Сегодня`, `Рутины`, `Цели` и `Прогресс` после прогона.

## Детерминированные Worker-Сценарии

Для проверок прогресса и напоминаний можно сбрасывать состояние, двигать часы и запускать
один шаг рабочего процесса через локальный control API:

```bash
curl -fsS -X POST 'http://127.0.0.1:8082/control/reset?chat_id=<bot-api-chat-id>'
curl -fsS -X POST 'http://127.0.0.1:8082/control/clock' \
  -H 'content-type: application/json' \
  -d '{"now":"2026-05-29T00:05:00Z"}'
curl -fsS -X POST 'http://127.0.0.1:8082/control/tick'
curl -fsS -X POST 'http://127.0.0.1:8082/control/clock' \
  -H 'content-type: application/json' \
  -d '{}'
```

Control API включается только вне production-окружения.

Сценарии `12` и `13`, завязанные на время, удобнее запускать частями: сначала пользовательская
настройка, затем `control/clock` + `control/tick`, затем оставшиеся шаги сценария.
Для `12` выставляй 08:00 следующего локального дня после настройки: карточка
отмечает предыдущую дату. Для reminder/auto-close проверки отдельно двигай часы
на 20:00 и 00:00 по локальному времени.
Иначе запускатель может начать ждать карточку, которая создается только шагом рабочего процесса.
Для вставки про цели в `14` перед запуском нужны уже сохраненные активные цели и дата,
которая проходит детерминированную проверку; в live-прогоне использовался
`2026-06-10T09:00:00Z`.

## Финальная Сверка Прогона

Не считай полный live E2E закрытым только по успешному коду выхода сценариев.
В конце каждого полного прогона проверь логи запускателя, состояние локальной БД и
последние логи приложения.
Также проверь снимок истории по затронутым темам: в чате должны остаться только
нужные итоговые сообщения, закрепы и служебные сообщения Telegram. Лишние prompt,
alert, draft, cleanup или user-input сообщения означают, что сценарий не закрыт.

```bash
cd /Users/igor/projects/trackmate

RUN_ID=$(cat tmp/e2e-current-run-id)
LOG_DIR="tmp/e2e-live-logs-$RUN_ID"

failed=0
for f in "$LOG_DIR"/*.log; do
  if rg -q '"type":"timeout"|"type":"error"|callback: read scenario|exit status|panic' "$f"; then
    echo "bad $f"
    failed=1
  fi
done
test "$failed" -eq 0
```

```bash
docker compose exec -T postgres psql -U postgres -d trackmate -Atc "
select 'pending_inputs', count(*) from pending_inputs
union all
select 'progress_unpublished', count(*) from progress_events where published_at is null;"
```

Ожидаемо: `pending_inputs=0`, неопубликованных progress-событий `0`. Последний
сценарий может оставить допустимое доменное состояние, например ограничение
повторной вставки про цели.

```bash
docker compose ps
docker compose logs --since=30m api worker | rg -i 'panic|fatal|error|warn|timeout' || true
```

Если E2E двигал локальные часы через `control/clock`, в конце верни системное
время для worker:

```bash
curl -fsS -X POST 'http://127.0.0.1:8082/control/clock' \
  -H 'content-type: application/json' \
  -d '{}'
```

Запиши id прогона, путь к логам, сводку БД и результат проверок в handoff или
сообщение для ревью.

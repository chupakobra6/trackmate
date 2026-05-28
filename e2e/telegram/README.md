# Telegram E2E

Здесь лежат только продуктовые сценарии Trackmate. Общий runner, MTProto-сессия,
локи, бинарные fixture-файлы и transcript-артефакты остаются в соседнем репозитории
`/Users/igor/projects/telegram-bot-e2e-test-tool`.

## Что Проверяем

- setup создает и чинит только темы `Сегодня` и `Прогресс`;
- задача дня создается только в теме `Сегодня`;
- второй открытый фокус для того же участника блокируется;
- отчеты закрывают задачу со статусами `выполнена`, `выполнена частично`,
  `не выполнена`;
- pending input не съедает сообщение из другой темы;
- фотоальбом в отчете закрывает задачу один раз;
- worker публикует событие в теме `Прогресс`;
- напоминание подтверждается кнопкой `👀 Понял`.

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
- `topic_id=` для `Прогресс` -> `PROGRESS_THREAD_ID`;
- `WRONG_THREAD_ID` можно поставить равным `PROGRESS_THREAD_ID` для проверки
  wrong-topic pending input.

## Рендер Темплейтов

```bash
export TRACKMATE_CHAT='3871708263'
export TODAY_THREAD_ID='10'
export PROGRESS_THREAD_ID='11'
export WRONG_THREAD_ID="$PROGRESS_THREAD_ID"

mkdir -p tmp/e2e-rendered
for src in e2e/telegram/scenarios/*.jsonl.tmpl; do
  dst="tmp/e2e-rendered/$(basename "${src%.tmpl}")"
  sed \
    -e "s/{{CHAT}}/$TRACKMATE_CHAT/g" \
    -e "s/{{TODAY_THREAD_ID}}/$TODAY_THREAD_ID/g" \
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
Сценарии не содержат `dump_state`: runner сам пишет transcript-артефакты, а
продуктовые файлы здесь описывают только проверяемый пользовательский путь.
Кнопки на карточках задач кликаются с `message_text`, чтобы в давно используемой
тестовой группе не попасть в старую кнопку из истории.

## Сценарии

- `00-setup-smoke.jsonl`: `/setup`, повторная проверка условий, оформление группы.
- `01-today-add-prompt.jsonl.tmpl`: кнопка добавления задачи.
- `02-today-create-task.jsonl.tmpl`: создание задачи дня.
- `03-today-block-second-task.jsonl.tmpl`: запрет второй задачи.
- `04-report-done.jsonl.tmpl`: отчет `✅ Выполнена`.
- `05-report-partial.jsonl.tmpl`: отчет `🔸 Выполнена частично`.
- `06-report-failed.jsonl.tmpl`: отчет `❌ Не выполнена`.
- `07-wrong-topic-pending-ignored.jsonl.tmpl`: pending input не закрывается
  сообщением из другой темы.
- `08-duplicate-photo-report-consumes-once.jsonl.tmpl`: Telegram album/media group
  из двух фото с общей подписью закрывает отчет один раз.
- `09-progress-topic-event.jsonl.tmpl`: событие появляется в `Прогресс`.
- `10-alert-ack.jsonl.tmpl`: кнопка `👀 Понял` подтверждает напоминание.

## Детерминированные Worker-Сценарии

Для progress и alert проверок можно сбрасывать состояние, двигать clock и запускать
один worker tick через локальный control API:

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

Control API включается только в non-production окружении.

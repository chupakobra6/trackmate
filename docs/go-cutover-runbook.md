# Перезапуск На Go Runtime

Этот runbook нужен для локальной проверки и будущего аккуратного перехода
боевого pet-проекта на Go-only runtime. Команды ниже не запускать против
production случайно: сначала проверь `.env`, текущий git commit и Docker context.

## Что Должно Сохраниться

Сохраняем:

- группы и настройки workspace;
- привязки тем `Сегодня` и `Прогресс`;
- участников;
- ежедневные задачи, отчеты и статусы;
- напоминания;
- progress history, кроме старых Materials-событий.

Удаляем намеренно:

- Python runtime, Alembic и pytest-слой;
- Materials topic binding в БД;
- material tables и material enum values;
- старые material progress events.

Сам Telegram topic `Материалы`, если он еще есть в группе, Go runtime не трогает.
Его можно удалить руками после проверки.

## Локальная Fresh-DB Проверка

```bash
cd /Users/igor/projects/trackmate
cp .env.example .env
make docker-reset
docker compose ps
docker compose logs --tail=200 migrate api worker
```

Ожидаемо:

- `migrate` завершился успешно и написал `migrations_applied`;
- `postgres`, `api`, `worker` healthy;
- Python-процессов для Trackmate нет.

## Проверка Миграции Старой БД

Для копии старой БД:

```bash
make docker-db-backup-stop
make docker-db-restore FILE=backups/<dump>.dump
docker compose run --rm migrate
```

Проверить сохраненные данные:

```sql
SELECT count(*) FROM workspace_groups;
SELECT count(*) FROM participants;
SELECT count(*) FROM daily_tasks;
SELECT count(*) FROM daily_task_alerts;
SELECT count(*) FROM progress_events;
```

Проверить, что Materials удален из схемы:

```sql
SELECT to_regclass('public.material_batches');
SELECT to_regclass('public.material_items');
SELECT to_regclass('public.material_participant_progresses');
SELECT enumlabel
FROM pg_enum e
JOIN pg_type t ON t.oid = e.enumtypid
WHERE t.typname = 'topickey';
SELECT enumlabel
FROM pg_enum e
JOIN pg_type t ON t.oid = e.enumtypid
WHERE t.typname = 'progresseventtype';
```

Ожидаемо:

- material tables возвращают `NULL`;
- `topickey` содержит только `today`, `progress`;
- `progresseventtype` содержит `daily_task.closed`, `daily_task.auto_failed`,
  `system_alert`, `custom_update`.

## Переезд На Сервере

1. Зафиксировать текущий рабочий commit и `.env`.

```bash
git rev-parse --short HEAD
docker compose ps
```

2. Сделать финальный dump и оставить старые `api`/`worker` остановленными.

```bash
make docker-db-backup-stop
```

После этого старый polling worker не запускать, иначе он снова начнет писать в БД.

3. Подтянуть Go-only код.

```bash
git pull --ff-only
```

4. Убедиться, что tracked Python runtime действительно исчез из рабочей копии.

```bash
test ! -e pyproject.toml
test ! -d src/trackmate
test ! -d alembic
```

5. Убрать локальные Python-кэши, если они остались как untracked файлы.

```bash
make clean-legacy
```

6. Остановить старые контейнеры без удаления PostgreSQL volume.

```bash
docker compose down --remove-orphans
```

Не использовать `docker compose down -v` на сервере: это удалит volume с БД.

7. Собрать и поднять Go stack.

```bash
docker compose up -d --build
docker compose ps
docker compose logs --tail=200 migrate api worker
```

`migrate` применит goose migrations перед стартом `api` и `worker`.

8. Проверить БД после миграций SQL-запросами из раздела выше.

9. Проверить Telegram:

- старая карточка `Сегодня` останется той же, если ее `control_message_id`
  был сохранен в `topic_bindings`;
- `/setup` можно нажать один раз для repair: Go чинит только `Сегодня` и
  `Прогресс`, Materials не создает;
- Progress intro будет создан или отредактирован только в теме `Прогресс`;
- migration `202605280004_trackmate_10_go_announcement.sql` создаст одно
  pending-событие `Встречайте: Trackmate 1.0 на Go`, worker опубликует его
  в `Прогресс`.

## Telegram E2E

Runner: `/Users/igor/projects/telegram-bot-e2e-test-tool`.

Минимальный набор:

```bash
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/e2e/telegram/scenarios/00-setup-smoke.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/02-today-create-task.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/04-report-done.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/07-wrong-topic-pending-ignored.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/08-duplicate-photo-report-consumes-once.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/09-progress-topic-event.jsonl
CHAT='<mtproto-chat-target>' go run ./cmd/tg-e2e-tool run-scenario /Users/igor/projects/trackmate/tmp/e2e-rendered/10-alert-ack.jsonl
```

Для deterministic progress/alert проверок используй local-only control API:

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

## Rollback

Rollback только через restore pre-cutover dump:

1. остановить `api` и `worker`;
2. восстановить dump;
3. если нужен откат runtime, запускать старый Python runtime только после
   осознанного решения.

Миграция удаления Materials намеренно необратима на уровне продукта.

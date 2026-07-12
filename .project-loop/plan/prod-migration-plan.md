# План Production-Миграции

Проект: Trackmate
Статус: одобрено пользователем для S030.

## Что Меняется В БД

Миграция: `migrations/202607120001_add_daily_summaries.sql`.

- создается enum `dailyentrykind`: `task`, `summary`;
- в `daily_tasks` добавляется обязательная `entry_kind` с default `task`,
  поэтому существующая история остается историей обычных задач;
- создается индекс `ix_daily_tasks_entry_kind`;
- в `progresseventtype` добавляются `daily_summary.closed` и
  `daily_summary.auto_failed`.

Миграция additive: она не удаляет, не переписывает и не переоценивает текущие
задачи, итоги, участников, напоминания или события прогресса.

## Риск И Откат

До появления первого production-итога дня старый бинарник совместим со схемой:
он игнорирует новую колонку и не создает новые event types. После появления
`daily_summary.*` откат приложения выполняется только через восстановление
backup или forward-fix, потому что старый formatter не знает эти события.

## Локальная Проверка

- local Docker migration `202607120001`: pass;
- clean-schema PostgreSQL `TRACKMATE_TEST_DATABASE_URL=... go test ./... -count=1`: pass;
- `make test`, `make lint`, `go vet ./...`, `git diff --check`: pass;
- live Telegram проверены нормальный итог, edit sync и auto-fail карточки;
- local worker/API/PostgreSQL healthy, outbox и pending inputs пусты.

## Production-Порядок

1. Сделать `make docker-db-backup-stop`.
2. Выполнить `git pull --ff-only` на нужный commit.
3. Запустить `docker compose up -d --build`; migrate-контейнер применяет
   `202607120001` до запуска API и worker.
4. Проверить health, логи, `dailyentrykind`, новые labels `progresseventtype`,
   сохранность counts, `pending_inputs=0` и пустой Progress outbox.
5. Запустить `/setup` или `setup:start`, чтобы Today и Progress control-сообщения
   обновились с новой подсказкой.

## Rollback

Если migration/build не проходят проверку до появления новых итогов, восстановить
backup через `make docker-db-restore FILE=<backup>`. Goose Down не удаляет enum
labels, чтобы не переписывать production columns.

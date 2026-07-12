# План Production-Миграции

Проект: Trackmate
Статус: выполнено на production.

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

## Production-Результат

1. Перед развёртыванием создан backup: `/opt/trackmate/backups/trackmate_20260712T203143Z.dump`.
2. На production развёрнут commit `848042d`; `docker compose up -d --build`
   применил миграцию `202607120001`.
3. `api`, `worker` и `postgres` находятся в состоянии healthy; error-log scan
   после развёртывания чистый.
4. Схема содержит `dailyentrykind` (`task`, `summary`) и новые события
   `daily_summary.closed`/`daily_summary.auto_failed`; существующие 201 записей
   остались типом `task`, количество данных до и после развёртывания совпало.
5. Пустой Progress outbox и неизменный `pending_inputs=3` подтверждены. Контрольные
   сообщения Today (`8`) и Progress (`10`) обновлены через Bot API и повторно
   сверены по тексту.

## Rollback

Если migration/build не проходят проверку до появления новых итогов, восстановить
backup через `make docker-db-restore FILE=<backup>`. Goose Down не удаляет enum
labels, чтобы не переписывать production columns.

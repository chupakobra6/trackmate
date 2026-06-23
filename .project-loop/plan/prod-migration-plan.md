# Черновой План Prod-Миграции

Проект: trackmate  
Статус: draft for review, production deploy requires explicit approval.

## Что Меняется В БД

Миграции:

- `migrations/202606230001_add_routines_and_goals.sql`;
- `migrations/202606230002_add_goal_nudge_cooldowns.sql`.

Операции:

- additive enum labels для `topickey`: `routine`, `goals`;
- новые enum types: `routineitemstatus`, `goalfinalstatus`;
- новые таблицы:
  - `routine_plans`;
  - `routine_checkins`;
  - `routine_checkin_items`;
  - `seasonal_goal_sets`;
  - `seasonal_goal_weekly_reviews`;
  - `seasonal_goal_final_reviews`;
  - `goal_nudge_cooldowns`;
- новые индексы на foreign keys, owners, dates, statuses.

Текущие таблицы `daily_tasks`, `daily_task_alerts`, `progress_events`, `participants`, `workspace_groups`, `pending_inputs` не удаляются и не переписываются.

## Риск Данных

Низкий для текущей истории: миграция не содержит `DELETE`, `UPDATE` существующей истории, `DROP` существующих product tables или изменения колонок existing history tables.

Отдельный риск: `ALTER TYPE topickey ADD VALUE` требует PostgreSQL-compatible migration path; файл помечен `-- +goose NO TRANSACTION`, чтобы не упереться в ограничения enum DDL. Enum `DO $$` блоки обернуты в `-- +goose StatementBegin/StatementEnd`, это проверено на fresh schema в локальном PostgreSQL.

## Локальный Dry-Run Уже Выполнен

- Docker compose проверен: `postgres`, `api`, `worker` подняты и healthy/up.
- `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./...`: pass.
- `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./... -cover`: pass.
- `make lint`: pass.
- `TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' make test`: pass.
- `TRACKMATE__DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' make migrate`: pass.
- `loopctl.py validate /Users/igor/projects/trackmate`: pass.

## Обязательный Dry-Run Перед Prod

1. Поднять локальную/test PostgreSQL.
2. Выполнить:

```bash
TRACKMATE_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go test ./...
```

3. Отдельно проверить миграцию на копии или disposable schema:

```bash
TRACKMATE__DATABASE_URL='postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable' go run ./cmd/migrate
```

4. Проверить, что existing rows остались:

```sql
select count(*) from daily_tasks;
select count(*) from progress_events;
select count(*) from participants;
select count(*) from topic_bindings;
```

## Prod-Порядок После Approval

1. Сделать backup production database штатным способом проекта.
2. Зафиксировать текущие counts для history/stat tables:
   - `daily_tasks`;
   - `progress_events`;
   - `daily_task_alerts`;
   - `participants`;
   - `workspace_groups`;
   - `topic_bindings`;
   - `pending_inputs`.
3. Остановить или временно заморозить `api`/`worker`, чтобы во время миграции не было конкурирующих writes.
4. Обновить код на production host.
5. Запустить goose migrations через `trackmate migrate` / `go run ./cmd/migrate` в production environment.
6. Повторить counts из пункта 2 и убедиться, что они не уменьшились.
7. Перезапустить `api` и `worker`.
8. Запустить `/setup` или `setup:start` в группе, чтобы создать/починить `Рутины` и `Цели`.
9. Проверить smoke:
   - `Сегодня` принимает новую цель-задачу дня;
   - `Рутины` показывает pinned `✏️ Настроить рутину`;
   - `Цели` показывает pinned `✏️ Настроить цели`;
   - `Прогресс` не получает routine events;
   - goal nudges появляются только у участников с активными целями и не чаще одного раза за 3 дня.

## Rollback

До реального использования новых тем самый безопасный rollback — откат к backup.

Goose Down удаляет только новые routine/goals tables и новые enum types, но намеренно оставляет enum labels `routine`/`goals` в `topickey`, потому что удаление enum labels в PostgreSQL требует переписывания зависимых колонок и рискованнее для production.

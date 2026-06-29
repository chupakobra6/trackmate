# Текущий Шаг

Проект: trackmate
Обновлено: 2026-06-29

## Активный Шаг
- id: `STEP-014`
- status: `готово`
- objective: Исправить production case Егора и закрепить schema-fix для topic-scoped pending.
- requirement IDs: `REQ-031`
- owned paths: `.project-loop/`, `migrations/202606290001_drop_legacy_pending_input_unique.sql`
- validation: production SQL/log verification: pass; `TRACKMATE_TEST_DATABASE_URL=... go test ./internal/storage/postgres ./internal/bot ./internal/app/pending`: pass; `git diff --check`: pass; `make test`: pass; `make lint`: pass; `TRACKMATE__DATABASE_URL=... make migrate`: pass; `loopctl.py validate`: pass
- done criteria: task `160` on production stores report message `3386` and status `done`; wrong auto-fail progress event removed without new chat publication; legacy pending constraint removed on production; local migration exists and validation passes.

## Фокус Ревью
- Не деплоить локальные routine fixes из STEP-012/STEP-013 в рамках этой ручной прод-правки.
- Не создавать новую публикацию в `Прогресс` при восстановлении старого отчета.
- Схема должна разрешать несколько pending inputs для одного пользователя в разных `message_thread_id`.

## Примечания
- Production logs 2026-06-24 19:33 UTC показали: `task:status:160:done` упал с `duplicate key value violates unique constraint "uq_pending_inputs_workspace_group_id"`, поэтому message `3386` не мог быть привязан к pending report.
- На production удален legacy constraint `uq_pending_inputs_workspace_group_id`; остался `ux_pending_inputs_workspace_user_thread`.
- Production task `160` исправлен на `done`, `report_message_id=3386`, `report_text=Голосовое сообщение`; неверный `progress_events.id=183` удален без новой публикации.
- Telegram не дал удалить старые сообщения `3385` и `3404`: `Bad Request: message can't be deleted`.

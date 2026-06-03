-- +goose Up
ALTER TABLE daily_tasks
    ADD COLUMN IF NOT EXISTS task_message_id INTEGER,
    ADD COLUMN IF NOT EXISTS task_message_thread_id INTEGER,
    ADD COLUMN IF NOT EXISTS report_message_id INTEGER,
    ADD COLUMN IF NOT EXISTS report_message_thread_id INTEGER;

CREATE UNIQUE INDEX IF NOT EXISTS ux_daily_tasks_task_message_source
    ON daily_tasks(workspace_group_id, task_message_id)
    WHERE task_message_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_daily_tasks_report_message_source
    ON daily_tasks(workspace_group_id, report_message_id)
    WHERE report_message_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS ux_daily_tasks_report_message_source;
DROP INDEX IF EXISTS ux_daily_tasks_task_message_source;

ALTER TABLE daily_tasks
    DROP COLUMN IF EXISTS report_message_thread_id,
    DROP COLUMN IF EXISTS report_message_id,
    DROP COLUMN IF EXISTS task_message_thread_id,
    DROP COLUMN IF EXISTS task_message_id;

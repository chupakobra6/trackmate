-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE dailyentrykind AS ENUM ('task', 'summary');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

ALTER TABLE daily_tasks
    ADD COLUMN IF NOT EXISTS entry_kind dailyentrykind NOT NULL DEFAULT 'task';

CREATE INDEX IF NOT EXISTS ix_daily_tasks_entry_kind ON daily_tasks(entry_kind);

ALTER TYPE progresseventtype ADD VALUE IF NOT EXISTS 'daily_summary.closed';
ALTER TYPE progresseventtype ADD VALUE IF NOT EXISTS 'daily_summary.auto_failed';

-- +goose Down
-- Enum values are intentionally retained: removing them would require rewriting
-- live progress history. The migration is additive and preserves prior entries.

-- +goose Up
ALTER TABLE routine_plans
    ADD COLUMN IF NOT EXISTS source_message_id INTEGER,
    ADD COLUMN IF NOT EXISTS source_message_thread_id INTEGER;

ALTER TABLE routine_checkins
    ADD COLUMN IF NOT EXISTS source_message_id INTEGER,
    ADD COLUMN IF NOT EXISTS source_message_thread_id INTEGER;

-- +goose Down
ALTER TABLE routine_checkins
    DROP COLUMN IF EXISTS source_message_thread_id,
    DROP COLUMN IF EXISTS source_message_id;

ALTER TABLE routine_plans
    DROP COLUMN IF EXISTS source_message_thread_id,
    DROP COLUMN IF EXISTS source_message_id;

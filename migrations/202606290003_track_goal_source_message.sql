-- +goose Up
ALTER TABLE seasonal_goal_sets
    ADD COLUMN IF NOT EXISTS source_message_id INTEGER,
    ADD COLUMN IF NOT EXISTS source_message_thread_id INTEGER;

-- +goose Down
ALTER TABLE seasonal_goal_sets
    DROP COLUMN IF EXISTS source_message_thread_id,
    DROP COLUMN IF EXISTS source_message_id;

-- +goose Up
ALTER TABLE seasonal_goal_weekly_reviews
    ADD COLUMN IF NOT EXISTS response_message_id INTEGER,
    ADD COLUMN IF NOT EXISTS response_message_thread_id INTEGER;

ALTER TABLE seasonal_goal_final_reviews
    ADD COLUMN IF NOT EXISTS summary_message_ids BIGINT[] NOT NULL DEFAULT '{}'::BIGINT[];

-- +goose Down
ALTER TABLE seasonal_goal_final_reviews
    DROP COLUMN IF EXISTS summary_message_ids;

ALTER TABLE seasonal_goal_weekly_reviews
    DROP COLUMN IF EXISTS response_message_thread_id,
    DROP COLUMN IF EXISTS response_message_id;

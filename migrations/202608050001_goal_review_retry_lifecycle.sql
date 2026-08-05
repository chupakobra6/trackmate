-- +goose Up
ALTER TABLE seasonal_goal_weekly_reviews
    ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS skipped_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS ix_goal_weekly_reviews_open
    ON seasonal_goal_weekly_reviews (goal_set_id, requested_at)
    WHERE response_text IS NULL AND skipped_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS ix_goal_weekly_reviews_open;

ALTER TABLE seasonal_goal_weekly_reviews
    DROP COLUMN IF EXISTS skipped_at,
    DROP COLUMN IF EXISTS reminder_sent_at;

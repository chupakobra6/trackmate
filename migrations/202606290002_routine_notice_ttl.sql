-- +goose Up
ALTER TABLE routine_checkins
    ADD COLUMN IF NOT EXISTS auto_close_notice_message_id INTEGER,
    ADD COLUMN IF NOT EXISTS auto_close_notice_sent_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS ix_routine_checkins_auto_close_notice_sent_at
    ON routine_checkins(auto_close_notice_sent_at);

-- +goose Down
DROP INDEX IF EXISTS ix_routine_checkins_auto_close_notice_sent_at;

ALTER TABLE routine_checkins
    DROP COLUMN IF EXISTS auto_close_notice_sent_at,
    DROP COLUMN IF EXISTS auto_close_notice_message_id;

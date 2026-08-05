-- +goose Up
ALTER TABLE daily_task_alerts
    ADD COLUMN IF NOT EXISTS dispatch_started_at TIMESTAMPTZ;

ALTER TABLE progress_events
    ADD COLUMN IF NOT EXISTS publish_started_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS ix_daily_task_alerts_claimable
    ON daily_task_alerts (dispatch_status, dispatch_started_at, id)
    WHERE acknowledged_at IS NULL AND dispatch_status IN ('pending', 'dispatching');

CREATE INDEX IF NOT EXISTS ix_progress_events_claimable
    ON progress_events (publish_status, publish_started_at, id)
    WHERE publish_status IN ('pending', 'publishing');

-- +goose Down
DROP INDEX IF EXISTS ix_progress_events_claimable;
DROP INDEX IF EXISTS ix_daily_task_alerts_claimable;

ALTER TABLE progress_events
    DROP COLUMN IF EXISTS publish_started_at;

ALTER TABLE daily_task_alerts
    DROP COLUMN IF EXISTS dispatch_started_at;

-- +goose Up
ALTER TABLE pending_inputs
    ADD COLUMN IF NOT EXISTS message_thread_id INTEGER;

UPDATE pending_inputs
SET message_thread_id = CASE
    WHEN payload ? 'thread_id' AND payload->>'thread_id' ~ '^-?[0-9]+$'
        THEN (payload->>'thread_id')::integer
    ELSE 0
END
WHERE message_thread_id IS NULL;

ALTER TABLE pending_inputs
    ALTER COLUMN message_thread_id SET DEFAULT 0,
    ALTER COLUMN message_thread_id SET NOT NULL;

ALTER TABLE pending_inputs
    DROP CONSTRAINT IF EXISTS pending_inputs_workspace_group_id_user_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS ux_pending_inputs_workspace_user_thread
    ON pending_inputs(workspace_group_id, user_id, message_thread_id);

CREATE INDEX IF NOT EXISTS ix_pending_inputs_message_thread_id
    ON pending_inputs(message_thread_id);

CREATE INDEX IF NOT EXISTS ix_pending_inputs_created_at
    ON pending_inputs(created_at);

ALTER TABLE routine_checkins
    ADD COLUMN IF NOT EXISTS reminder_message_id INTEGER,
    ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS auto_failed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS ix_routine_checkins_completed_at
    ON routine_checkins(completed_at);

CREATE INDEX IF NOT EXISTS ix_routine_checkins_reminder_sent_at
    ON routine_checkins(reminder_sent_at);

-- +goose Down
DROP INDEX IF EXISTS ix_routine_checkins_reminder_sent_at;
DROP INDEX IF EXISTS ix_routine_checkins_completed_at;

ALTER TABLE routine_checkins
    DROP COLUMN IF EXISTS auto_failed_at,
    DROP COLUMN IF EXISTS reminder_sent_at,
    DROP COLUMN IF EXISTS reminder_message_id;

DROP INDEX IF EXISTS ix_pending_inputs_created_at;
DROP INDEX IF EXISTS ix_pending_inputs_message_thread_id;
DROP INDEX IF EXISTS ux_pending_inputs_workspace_user_thread;

ALTER TABLE pending_inputs
    ADD CONSTRAINT pending_inputs_workspace_group_id_user_id_key UNIQUE (workspace_group_id, user_id);

ALTER TABLE pending_inputs
    DROP COLUMN IF EXISTS message_thread_id;

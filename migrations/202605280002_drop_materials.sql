-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    has_material_topic boolean;
    has_material_progress boolean;
BEGIN
    IF to_regclass('pending_inputs') IS NOT NULL THEN
        DELETE FROM pending_inputs
        WHERE kind IN ('material_note', 'material_applied');
    END IF;

    IF to_regclass('progress_events') IS NOT NULL THEN
        DELETE FROM progress_events
        WHERE event_type::text IN ('material_note_added', 'material_applied');

        DROP INDEX IF EXISTS ix_progress_events_material_batch_id;
        ALTER TABLE progress_events DROP COLUMN IF EXISTS material_batch_id;
    END IF;

    DROP TABLE IF EXISTS material_participant_progresses CASCADE;
    DROP TABLE IF EXISTS material_items CASCADE;
    DROP TABLE IF EXISTS material_batches CASCADE;

    IF to_regclass('topic_bindings') IS NOT NULL THEN
        DELETE FROM topic_bindings
        WHERE topic_key::text = 'materials';
    END IF;

    SELECT EXISTS (
        SELECT 1
        FROM pg_type t
        JOIN pg_enum e ON e.enumtypid = t.oid
        WHERE t.typname = 'topickey' AND e.enumlabel = 'materials'
    ) INTO has_material_topic;

    IF has_material_topic AND to_regclass('topic_bindings') IS NOT NULL THEN
        ALTER TABLE topic_bindings ALTER COLUMN topic_key TYPE text USING topic_key::text;
        DROP TYPE topickey;
        CREATE TYPE topickey AS ENUM ('today', 'progress');
        ALTER TABLE topic_bindings ALTER COLUMN topic_key TYPE topickey USING topic_key::topickey;
    END IF;

    SELECT EXISTS (
        SELECT 1
        FROM pg_type t
        JOIN pg_enum e ON e.enumtypid = t.oid
        WHERE t.typname = 'progresseventtype' AND e.enumlabel IN ('material_note_added', 'material_applied')
    ) INTO has_material_progress;

    IF has_material_progress AND to_regclass('progress_events') IS NOT NULL THEN
        ALTER TABLE progress_events ALTER COLUMN event_type TYPE text USING event_type::text;
        DROP TYPE progresseventtype;
        CREATE TYPE progresseventtype AS ENUM ('daily_task.closed', 'daily_task.auto_failed', 'system_alert', 'custom_update');
        ALTER TABLE progress_events ALTER COLUMN event_type TYPE progresseventtype USING event_type::progresseventtype;
    END IF;

    DROP TYPE IF EXISTS materialhigheststate;
    DROP TYPE IF EXISTS materialbatchstatus;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Materials removal is intentionally irreversible. The product no longer stores
-- material batches, material progress, or material topic bindings.

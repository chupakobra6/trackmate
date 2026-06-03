-- +goose Up
ALTER TABLE progress_events
    ALTER COLUMN payload TYPE JSONB
    USING COALESCE(payload::jsonb, '{}'::jsonb);

ALTER TABLE pending_inputs
    ALTER COLUMN payload TYPE JSONB
    USING COALESCE(payload::jsonb, '{}'::jsonb);

-- +goose Down
ALTER TABLE pending_inputs
    ALTER COLUMN payload TYPE JSON
    USING COALESCE(payload::json, '{}'::json);

ALTER TABLE progress_events
    ALTER COLUMN payload TYPE JSON
    USING COALESCE(payload::json, '{}'::json);

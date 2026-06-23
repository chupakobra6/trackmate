-- +goose Up
CREATE TABLE IF NOT EXISTS goal_nudge_cooldowns (
    workspace_group_id INTEGER NOT NULL REFERENCES workspace_groups(id) ON DELETE CASCADE,
    participant_id INTEGER NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    last_shown_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_group_id, participant_id)
);
CREATE INDEX IF NOT EXISTS ix_goal_nudge_cooldowns_last_shown_at ON goal_nudge_cooldowns(last_shown_at);

-- +goose Down
DROP TABLE IF EXISTS goal_nudge_cooldowns;

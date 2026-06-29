-- +goose Up
ALTER TABLE pending_inputs
    DROP CONSTRAINT IF EXISTS uq_pending_inputs_workspace_group_id;

-- +goose Down
ALTER TABLE pending_inputs
    ADD CONSTRAINT uq_pending_inputs_workspace_group_id UNIQUE (workspace_group_id, user_id);

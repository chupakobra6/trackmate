-- +goose Up
CREATE TABLE monthly_reports (
    workspace_group_id INTEGER NOT NULL REFERENCES workspace_groups(id) ON DELETE CASCADE,
    month DATE NOT NULL CHECK (extract(day FROM month) = 1),
    status TEXT NOT NULL CHECK (status IN ('publishing', 'published', 'uncertain')),
    content_hash TEXT NOT NULL,
    text TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    thread_id BIGINT NOT NULL,
    message_id BIGINT,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    last_error TEXT,
    PRIMARY KEY (workspace_group_id, month)
);

-- +goose Down
DROP TABLE monthly_reports;

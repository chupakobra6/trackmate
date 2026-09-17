-- +goose Up
CREATE TABLE worker_delivery_retries (
    operation text NOT NULL,
    entity_id bigint NOT NULL,
    attempts integer NOT NULL CHECK (attempts BETWEEN 1 AND 8),
    next_attempt_at timestamptz NOT NULL,
    exhausted boolean NOT NULL DEFAULT false,
    last_error text NOT NULL,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (operation, entity_id)
);

-- +goose Down
DROP TABLE worker_delivery_retries;

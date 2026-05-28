-- +goose Up
DROP TABLE IF EXISTS alembic_version;

-- +goose Down
-- The Go runtime does not use Alembic metadata.

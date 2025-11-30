-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS metrics (
    id TEXT PRIMARY KEY,
    mtype TEXT NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS metrics;
-- +goose StatementEnd

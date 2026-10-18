-- +goose Up
CREATE TABLE IF NOT EXISTS metrics (
    id TEXT NOT NULL,
    mtype TEXT NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    PRIMARY KEY (id, mtype)
);

-- +goose Down
DROP TABLE IF EXISTS metrics;

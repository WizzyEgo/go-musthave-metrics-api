-- +goose Up
CREATE TABLE IF NOT EXISTS metrics (
    metric_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    id VARCHAR(255) NOT NULL,
    mtype VARCHAR(16) NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    UNIQUE (id, mtype)
);

-- +goose Down
DROP TABLE IF EXISTS metrics;

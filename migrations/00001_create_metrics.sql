CREATE TABLE IF NOT EXISTS metrics (
    id TEXT NOT NULL,
    mtype TEXT NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    PRIMARY KEY (id, mtype)
);

DROP TABLE IF EXISTS metrics;
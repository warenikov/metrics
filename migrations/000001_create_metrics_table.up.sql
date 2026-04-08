CREATE TABLE IF NOT EXISTS metrics (
    id    TEXT NOT NULL,
    mtype TEXT NOT NULL CHECK (mtype IN ('gauge', 'counter')),
    delta BIGINT,
    value DOUBLE PRECISION,
    PRIMARY KEY (id, mtype)
);

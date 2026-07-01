-- @ai-modified 2026-07-02 create sessions table for scs pgxstore
CREATE TABLE sessions (
    token  TEXT PRIMARY KEY,
    data   BYTEA       NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

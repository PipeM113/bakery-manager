-- Failed login attempts, to lock an e-mail after too many (docs/acceso.md). They live in the
-- database because every serverless instance is a different process. Only the normalized e-mail
-- and the moment are kept: never the password, never an address.
CREATE TABLE login_attempts (
    id           BIGSERIAL PRIMARY KEY,
    email        TEXT NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_login_attempts_email_time ON login_attempts (email, attempted_at);
ALTER TABLE login_attempts ENABLE ROW LEVEL SECURITY;

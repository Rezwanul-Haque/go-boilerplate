CREATE TABLE IF NOT EXISTS users (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                  VARCHAR(255) UNIQUE NOT NULL,
    password_hash          VARCHAR(255) NOT NULL,
    reset_token            VARCHAR(255),
    reset_token_expires_at TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_reset_token ON users(reset_token);

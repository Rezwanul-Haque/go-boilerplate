CREATE TABLE IF NOT EXISTS tinyurl (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    short_code           TEXT NOT NULL,
    original_url         TEXT NOT NULL,
    click_count          BIGINT NOT NULL,
    expires_at           TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

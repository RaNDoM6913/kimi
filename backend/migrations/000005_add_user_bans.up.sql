CREATE TABLE IF NOT EXISTS user_bans (
    user_id UUID PRIMARY KEY,
    banned BOOLEAN NOT NULL,
    reason TEXT NULL,
    updated_by_tg_id BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_bans_banned_updated_at
    ON user_bans (banned, updated_at DESC);

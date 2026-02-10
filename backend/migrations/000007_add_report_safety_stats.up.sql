ALTER TABLE reports
ADD COLUMN IF NOT EXISTS reporter_trust_score INTEGER NOT NULL DEFAULT 0;

ALTER TABLE reports
ADD COLUMN IF NOT EXISTS reporter_role TEXT NOT NULL DEFAULT 'user';

CREATE INDEX IF NOT EXISTS idx_reports_target_created
    ON reports(target_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS user_safety_stats (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    reports_24h INTEGER NOT NULL DEFAULT 0,
    reports_7d INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_safety_stats_updated_at
    ON user_safety_stats(updated_at DESC);

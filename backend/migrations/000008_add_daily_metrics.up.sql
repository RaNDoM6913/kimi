CREATE TABLE IF NOT EXISTS daily_metrics (
    day_key DATE NOT NULL,
    city_id TEXT NOT NULL,
    gender TEXT NOT NULL,
    looking_for TEXT NOT NULL,
    likes INTEGER NOT NULL DEFAULT 0,
    dislikes INTEGER NOT NULL DEFAULT 0,
    superlikes INTEGER NOT NULL DEFAULT 0,
    matches INTEGER NOT NULL DEFAULT 0,
    reports INTEGER NOT NULL DEFAULT 0,
    approved INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (day_key, city_id, gender, looking_for)
);

CREATE INDEX IF NOT EXISTS idx_daily_metrics_day_city
    ON daily_metrics(day_key, city_id);

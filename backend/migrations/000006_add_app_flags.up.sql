CREATE TABLE IF NOT EXISTS app_flags (
    key TEXT PRIMARY KEY,
    value_bool BOOLEAN NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_tg_id BIGINT NULL
);

INSERT INTO app_flags (key, value_bool, updated_at, updated_by_tg_id)
VALUES ('registration_enabled', TRUE, NOW(), NULL)
ON CONFLICT (key) DO NOTHING;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS bot_users (
    tg_id BIGINT PRIMARY KEY,
    username TEXT NULL,
    first_name TEXT NULL,
    last_name TEXT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS bot_roles (
    tg_id BIGINT PRIMARY KEY REFERENCES bot_users(tg_id),
    role TEXT NOT NULL,
    granted_by BIGINT NULL,
    granted_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS bot_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_tg_id BIGINT NOT NULL,
    action TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bot_audit_actor_created_at
    ON bot_audit (actor_tg_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bot_audit_action_created_at
    ON bot_audit (action, created_at DESC);

CREATE TABLE IF NOT EXISTS bot_moderation_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_tg_id BIGINT NOT NULL,
    actor_role TEXT NOT NULL,
    target_user_id UUID NOT NULL,
    moderation_item_id UUID NULL,
    decision TEXT NOT NULL,
    reason_code TEXT NULL,
    duration_sec INT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bot_moderation_actions_actor_created_at
    ON bot_moderation_actions (actor_tg_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bot_moderation_actions_decision_created_at
    ON bot_moderation_actions (decision, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bot_moderation_actions_target_user_id
    ON bot_moderation_actions (target_user_id);

CREATE TABLE IF NOT EXISTS bot_lookup_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_tg_id BIGINT NOT NULL,
    actor_role TEXT NOT NULL,
    query TEXT NOT NULL,
    found_user_id UUID NULL,
    action TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bot_lookup_actions_actor_created_at
    ON bot_lookup_actions (actor_tg_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bot_lookup_actions_action_created_at
    ON bot_lookup_actions (action, created_at DESC);

CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_event_id TEXT,
    idempotency_key TEXT NOT NULL,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL DEFAULT 'BYN',
    product_sku TEXT NOT NULL,
    status TEXT NOT NULL,
    result_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (provider IN ('tg_stars', 'external')),
    CHECK (status IN ('PENDING', 'SUCCEEDED', 'FAILED'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_transactions_idempotency
    ON payment_transactions(idempotency_key);

CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_transactions_provider_event
    ON payment_transactions(provider, provider_event_id)
    WHERE provider_event_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_payment_transactions_user_created
    ON payment_transactions(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS user_entitlements (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    plus_active_until TIMESTAMPTZ,
    incognito_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_credits (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    superlike_credits INTEGER NOT NULL DEFAULT 0,
    boost_credits INTEGER NOT NULL DEFAULT 0,
    message_wo_match_credits INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

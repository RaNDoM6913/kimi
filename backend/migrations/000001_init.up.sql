-- Core users and private data
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT NOT NULL UNIQUE,
    username TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_private (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    phone_e164 TEXT,
    phone_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_private_phone_hash ON user_private(phone_hash);

-- Profile and onboarding
CREATE TABLE IF NOT EXISTS profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    birthdate DATE,
    age SMALLINT,
    gender TEXT NOT NULL DEFAULT 'unknown',
    looking_for TEXT NOT NULL DEFAULT 'all',
    occupation TEXT,
    education TEXT,
    height_cm SMALLINT,
    eye_color TEXT,
    zodiac TEXT,
    languages TEXT[] NOT NULL DEFAULT '{}',
    goals TEXT[] NOT NULL DEFAULT '{}',
    profile_completed BOOLEAN NOT NULL DEFAULT FALSE,
    city_id TEXT,
    city TEXT,
    last_geo_at TIMESTAMPTZ,
    last_lat DOUBLE PRECISION,
    last_lon DOUBLE PRECISION,
    lat DOUBLE PRECISION,
    lon DOUBLE PRECISION,
    radius_km SMALLINT NOT NULL DEFAULT 3,
    age_min SMALLINT NOT NULL DEFAULT 18,
    age_max SMALLINT NOT NULL DEFAULT 30,
    has_circle BOOLEAN NOT NULL DEFAULT FALSE,
    photos_count SMALLINT NOT NULL DEFAULT 0,
    reports_count INTEGER NOT NULL DEFAULT 0,
    approved BOOLEAN NOT NULL DEFAULT FALSE,
    moderation_status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_profiles_city_gender_looking_for
    ON profiles(city_id, gender, looking_for);
CREATE INDEX IF NOT EXISTS idx_profiles_age_range ON profiles(age_min, age_max);
CREATE INDEX IF NOT EXISTS idx_profiles_feed_approved_created
    ON profiles(approved, city_id, created_at DESC, user_id DESC);

-- Media and moderation queue
CREATE TABLE IF NOT EXISTS media (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    s3_key TEXT NOT NULL UNIQUE,
    position SMALLINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, kind, position)
);

CREATE INDEX IF NOT EXISTS idx_media_user_status ON media(user_id, status);

CREATE TABLE IF NOT EXISTS moderation_items (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,
    target_id BIGINT,
    status TEXT NOT NULL DEFAULT 'PENDING',
    eta_bucket TEXT NOT NULL DEFAULT 'up_to_10',
    reason_text TEXT,
    required_fix_step TEXT,
    moderator_tg_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_moderation_items_status_eta
    ON moderation_items(status, eta_bucket, created_at DESC);

-- Swipe/like/match graph
CREATE TABLE IF NOT EXISTS swipes (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_swipes_actor_created
    ON swipes(actor_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_swipes_target_created
    ON swipes(target_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_swipes_actor_target_created
    ON swipes(actor_user_id, target_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS dislikes_state (
    actor_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hide_until TIMESTAMPTZ,
    never_show BOOLEAN NOT NULL DEFAULT FALSE,
    dislike_count INTEGER NOT NULL DEFAULT 0,
    until_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (actor_user_id, target_user_id)
);

CREATE INDEX IF NOT EXISTS idx_dislikes_state_actor_until
    ON dislikes_state(actor_user_id, until_at DESC);
CREATE INDEX IF NOT EXISTS idx_dislikes_state_actor_hide_until
    ON dislikes_state(actor_user_id, hide_until DESC);
CREATE INDEX IF NOT EXISTS idx_dislikes_state_actor_never_show
    ON dislikes_state(actor_user_id, never_show);
CREATE INDEX IF NOT EXISTS idx_dislikes_state_target
    ON dislikes_state(target_user_id);

CREATE TABLE IF NOT EXISTS likes (
    id BIGSERIAL PRIMARY KEY,
    from_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_super_like BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(from_user_id, to_user_id)
);

CREATE INDEX IF NOT EXISTS idx_likes_to_created ON likes(to_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_likes_from_created ON likes(from_user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS likes_reveals (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    liker_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, liker_user_id)
);

CREATE INDEX IF NOT EXISTS idx_likes_reveals_user_created
    ON likes_reveals(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS matches (
    id BIGSERIAL PRIMARY KEY,
    user_a_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active',
    tg_dm_link_a TEXT,
    tg_dm_link_b TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (user_a_id < user_b_id),
    UNIQUE(user_a_id, user_b_id)
);

CREATE INDEX IF NOT EXISTS idx_matches_user_a_created
    ON matches(user_a_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_matches_user_b_created
    ON matches(user_b_id, created_at DESC);

CREATE TABLE IF NOT EXISTS blocks (
    id BIGSERIAL PRIMARY KEY,
    actor_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(actor_user_id, target_user_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_target ON blocks(target_user_id);
CREATE INDEX IF NOT EXISTS idx_blocks_actor_target ON blocks(actor_user_id, target_user_id);

CREATE TABLE IF NOT EXISTS reports (
    id BIGSERIAL PRIMARY KEY,
    reporter_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    details TEXT,
    status TEXT NOT NULL DEFAULT 'new',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reports_target_status
    ON reports(target_user_id, status, created_at DESC);

-- Monetization and quotas
CREATE TABLE IF NOT EXISTS entitlements (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    plus_expires_at TIMESTAMPTZ,
    boost_until TIMESTAMPTZ,
    superlike_credits INTEGER NOT NULL DEFAULT 0,
    reveal_credits INTEGER NOT NULL DEFAULT 0,
    message_wo_match_credits INTEGER NOT NULL DEFAULT 0,
    like_tokens INTEGER NOT NULL DEFAULT 0,
    incognito_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quotas_daily (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day_key DATE NOT NULL,
    tz_name TEXT NOT NULL DEFAULT 'UTC',
    likes_used INTEGER NOT NULL DEFAULT 0,
    rewind_used INTEGER NOT NULL DEFAULT 0,
    too_fast_hits INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, day_key)
);

CREATE INDEX IF NOT EXISTS idx_quotas_daily_day_key ON quotas_daily(day_key);

CREATE TABLE IF NOT EXISTS purchases (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sku TEXT NOT NULL,
    provider TEXT NOT NULL,
    external_txn_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    amount_minor BIGINT,
    currency TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchases_user_status_created
    ON purchases(user_id, status, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS uq_purchases_provider_external_txn
    ON purchases(provider, external_txn_id)
    WHERE external_txn_id IS NOT NULL;

-- Ads, partners, analytics
CREATE TABLE IF NOT EXISTS partners (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'offer',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ads (
    id BIGSERIAL PRIMARY KEY,
    partner_id BIGINT REFERENCES partners(id) ON DELETE SET NULL,
    title TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL DEFAULT 'IMAGE',
    asset_url TEXT NOT NULL,
    click_url TEXT NOT NULL,
    city_id TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    priority INTEGER NOT NULL DEFAULT 0,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (kind IN ('IMAGE', 'VIDEO'))
);

CREATE INDEX IF NOT EXISTS idx_ads_active_city_kind ON ads(is_active, city_id, kind);

CREATE TABLE IF NOT EXISTS ad_events (
    id BIGSERIAL PRIMARY KEY,
    ad_id BIGINT NOT NULL REFERENCES ads(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ad_events_ad_event_created
    ON ad_events(ad_id, event_type, created_at DESC);

CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_user_occurred
    ON events(user_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_name_occurred
    ON events(name, occurred_at DESC);

-- Optional SQL-side session audit (session source of truth is Redis)
CREATE TABLE IF NOT EXISTS session_audit (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    sid TEXT NOT NULL,
    action TEXT NOT NULL,
    ip TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_session_audit_user_created
    ON session_audit(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_session_audit_sid_created
    ON session_audit(sid, created_at DESC);

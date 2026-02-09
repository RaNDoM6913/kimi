ALTER TABLE likes
ADD COLUMN IF NOT EXISTS is_suspect BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_likes_is_suspect_created
    ON likes(is_suspect, created_at DESC);

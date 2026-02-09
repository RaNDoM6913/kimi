ALTER TABLE moderation_items
    ADD COLUMN IF NOT EXISTS locked_by_tg_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_moderation_items_pending_fifo_lock
    ON moderation_items (status, created_at ASC, id ASC, locked_until);

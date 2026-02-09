DROP INDEX IF EXISTS idx_moderation_items_pending_fifo_lock;

ALTER TABLE moderation_items
    DROP COLUMN IF EXISTS locked_at,
    DROP COLUMN IF EXISTS locked_until,
    DROP COLUMN IF EXISTS locked_by_tg_id;

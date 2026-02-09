ALTER TABLE moderation_items
    DROP COLUMN IF EXISTS decided_at,
    DROP COLUMN IF EXISTS reason_code;

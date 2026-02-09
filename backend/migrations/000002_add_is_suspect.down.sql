DROP INDEX IF EXISTS idx_likes_is_suspect_created;

ALTER TABLE likes
DROP COLUMN IF EXISTS is_suspect;

-- Rollback migration for ownership columns

ALTER TABLE albums DROP COLUMN IF EXISTS user_id;
ALTER TABLE media DROP COLUMN IF EXISTS user_id;

DROP INDEX IF EXISTS idx_albums_user_id;
DROP INDEX IF EXISTS idx_media_user_id;

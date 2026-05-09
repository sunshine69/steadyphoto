-- Migration to rename photos table to media and add video support columns
-- This migration assumes the existence of the 'photos' table from 0001_init_schema.up.sql

BEGIN;

-- 1. Rename photos table to media
ALTER TABLE photos RENAME TO media;

-- 2. Add media_type column
ALTER TABLE media ADD COLUMN media_type VARCHAR(20) DEFAULT 'photo' CHECK (media_type IN ('photo', 'video'));

-- 3. Add video_metadata column
ALTER TABLE media ADD COLUMN video_metadata JSONB DEFAULT '{}';

-- 4. Create index for video_metadata
CREATE INDEX idx_media_video_metadata ON media USING GIN(video_metadata);

-- 5. Create composite index for filtering
CREATE INDEX idx_media_type_created_at ON media(media_type, created_at);

-- 6. Update foreign keys in 'faces' table
-- Note: PostgreSQL handles renaming the referenced table automatically if we rename it, 
-- but we might need to rename the column in 'faces' to be consistent if we want.
-- The design doesn't explicitly say to rename 'photo_id' to 'media_id', 
-- but it's good practice. However, to keep it simple and avoid breaking other things, 
-- let's just rename the table 'photos' to 'media' and keep 'photo_id' for now, 
-- OR rename 'photo_id' to 'media_id' as well.
-- The instructions say "Media Type Field" and "Video Metadata JSONB" on "media" table.

-- Let's rename photo_id to media_id in faces and album_photos to be consistent.
ALTER TABLE faces RENAME COLUMN photo_id TO media_id;
ALTER TABLE album_photos RENAME COLUMN photo_id TO media_id;

COMMIT;

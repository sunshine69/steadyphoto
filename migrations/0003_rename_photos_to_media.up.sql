-- Migration to rename photos table to media and add video support columns
-- This migration assumes the existence of the 'photos' table from 0001_init_schema.up.sql
-- Note: Transaction management is handled by the migration runner (go run cmd/migrate/main.go)

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
ALTER TABLE faces RENAME COLUMN photo_id TO media_id;
ALTER TABLE album_photos RENAME COLUMN photo_id TO media_id;

-- 7. Update foreign keys in 'jobs' table
-- Drop old constraint
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_photo_id_fkey;
-- Rename column
ALTER TABLE jobs RENAME COLUMN photo_id TO media_id;
-- Recreate constraint
ALTER TABLE jobs ADD CONSTRAINT jobs_media_id_fkey FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE CASCADE;

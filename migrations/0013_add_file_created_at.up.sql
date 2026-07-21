-- Add file_created_at column to store filesystem creation time from device
ALTER TABLE media ADD COLUMN IF NOT EXISTS file_created_at TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_media_items_file_created_at ON media(file_created_at);

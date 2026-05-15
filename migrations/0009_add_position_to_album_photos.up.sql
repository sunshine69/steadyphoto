ALTER TABLE album_photos ADD COLUMN IF NOT EXISTS position INT DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_album_photos_position ON album_photos(album_id, position);

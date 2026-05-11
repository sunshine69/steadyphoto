ALTER TABLE media ADD COLUMN tags TEXT DEFAULT '';
CREATE INDEX idx_media_tags ON media (tags);

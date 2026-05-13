-- Migration to add ownership (user_id) to media and albums
-- This allows us to isolate data per user in a multi-tenant environment.

-- 1. Add user_id to 'media' table
ALTER TABLE media ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;

-- 2. Add user_id to 'albums' table
ALTER TABLE albums ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;

-- 3. Create indexes for faster ownership-based lookups
CREATE INDEX IF NOT EXISTS idx_media_user_id ON media(user_id);
CREATE INDEX IF NOT EXISTS idx_albums_user_id ON albums(user_id);

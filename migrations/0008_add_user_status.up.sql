-- Add status column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';

-- Ensure active is the default if we are adding it as a new column in an existing DB with data
UPDATE users SET status = 'active' WHERE status IS NULL;

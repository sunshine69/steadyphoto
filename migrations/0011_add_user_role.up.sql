-- Migration to add user role column and update status values

-- Add role column (default 'user' for existing users)
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'user';

-- Set the admin user's role to 'admin' (seeded in migration 0007)
UPDATE users SET role = 'admin' WHERE email = 'admin@steadyphoto.com';

-- Update pending/rejected status support
UPDATE users SET status = 'pending' WHERE status IS NULL OR status = '';

-- Create index on status and role for admin queries
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

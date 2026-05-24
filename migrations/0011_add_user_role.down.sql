-- Down migration: remove role column and indexes

DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_role;

ALTER TABLE users DROP COLUMN IF EXISTS role;

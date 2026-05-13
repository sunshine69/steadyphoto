-- Rollback migration for users and sessions

DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS users;

-- 000002_create_users.down.sql
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;

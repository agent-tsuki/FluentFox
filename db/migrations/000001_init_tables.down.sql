-- db/migrations/000001_create_users_table.down.sql
DROP TABLE IF EXISTS users_profile;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS jlpt_level;
DROP TYPE IF EXISTS users_settings;
DROP TYPE IF EXISTS user_verification;

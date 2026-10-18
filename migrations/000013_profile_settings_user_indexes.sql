-- +goose Up
-- Every profile, settings and login read looks these rows up by user_id;
-- without an index each one scans the whole table.
CREATE INDEX IF NOT EXISTS idx_users_profile_user_id ON users_profile (user_id);
CREATE INDEX IF NOT EXISTS idx_users_settings_user_id ON users_settings (user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_users_settings_user_id;
DROP INDEX IF EXISTS idx_users_profile_user_id;

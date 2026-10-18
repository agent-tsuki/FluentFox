-- +goose Up
-- One refresh_tokens row per signed-in device. The old one-row-per-user index
-- made a login on one device silently sign out every other device.
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;

-- Device and location are captured at login for the profile's device list.
-- Only a masked IP prefix is kept; latitude/longitude are rounded by the app.
ALTER TABLE refresh_tokens
    ADD COLUMN user_agent          text,
    ADD COLUMN ip_prefix           text,
    ADD COLUMN country_code        text,
    ADD COLUMN region              text,
    ADD COLUMN city                text,
    ADD COLUMN latitude            double precision,
    ADD COLUMN longitude           double precision,
    ADD COLUMN location_source     text,
    ADD COLUMN location_accuracy_m integer,
    ADD COLUMN is_vpn              boolean,
    ADD COLUMN last_used_at        timestamptz NOT NULL DEFAULT now(),
    ADD CONSTRAINT chk_refresh_tokens_user_agent_len CHECK (char_length(user_agent) <= 512),
    ADD CONSTRAINT chk_refresh_tokens_location_source CHECK (location_source IN ('ip', 'gps'));

CREATE INDEX idx_refresh_tokens_user_active ON refresh_tokens (user_id, last_used_at DESC) WHERE is_revoked = false;
-- Replaces the dropped unique index for lookups that include revoked rows:
-- pruning at login and revoking all sessions on password reset.
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id);

ALTER TABLE users ADD COLUMN last_login_at timestamptz;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;
DROP INDEX IF EXISTS idx_refresh_tokens_user;
DROP INDEX IF EXISTS idx_refresh_tokens_user_active;
ALTER TABLE refresh_tokens
    DROP CONSTRAINT IF EXISTS chk_refresh_tokens_location_source,
    DROP CONSTRAINT IF EXISTS chk_refresh_tokens_user_agent_len,
    DROP COLUMN IF EXISTS last_used_at,
    DROP COLUMN IF EXISTS is_vpn,
    DROP COLUMN IF EXISTS location_accuracy_m,
    DROP COLUMN IF EXISTS location_source,
    DROP COLUMN IF EXISTS longitude,
    DROP COLUMN IF EXISTS latitude,
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS region,
    DROP COLUMN IF EXISTS country_code,
    DROP COLUMN IF EXISTS ip_prefix,
    DROP COLUMN IF EXISTS user_agent;

-- Going back to one row per user: keep each user's newest session.
DELETE FROM refresh_tokens older
USING refresh_tokens newer
WHERE older.user_id = newer.user_id
  AND (older.updated_at, older.id) < (newer.updated_at, newer.id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens (user_id);

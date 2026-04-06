-- 000004_create_user_settings.up.sql
-- Per-user application preferences, UI toggles, and notification settings.
-- Separated from profiles because these are configuration values, not identity.
-- One row per user.

CREATE TABLE IF NOT EXISTS user_settings (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID        NOT NULL,

    -- UI preferences
    cursor_trail         BOOLEAN     NOT NULL DEFAULT TRUE,
    background_animation BOOLEAN     NOT NULL DEFAULT TRUE,

    -- Notification preferences
    email_digest         BOOLEAN     NOT NULL DEFAULT TRUE,
    daily_reminder       BOOLEAN     NOT NULL DEFAULT TRUE,
    -- Local time to send the daily reminder. NULL = use platform default (08:00)
    reminder_time        TIME        NULL,

    -- IANA timezone identifier. Critical for streak calculation.
    -- Sent by the browser on registration: Intl.DateTimeFormat().resolvedOptions().timeZone
    timezone             VARCHAR(50) NOT NULL DEFAULT 'UTC',

    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_settings_user_id  UNIQUE      (user_id),
    CONSTRAINT fk_user_settings_users    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_settings_user_id ON user_settings (user_id);

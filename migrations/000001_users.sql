-- The user tables as GORM AutoMigrate built them from internal/users/models.go.
-- Every statement uses IF NOT EXISTS so this is a no-op (apart from the
-- refresh-token dedupe) on databases AutoMigrate already created.
-- user_id columns are text, not uuid, and have no foreign keys, because that is
-- what AutoMigrate produced; changing them is a separate migration.

-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id                uuid NOT NULL,
    username          text NOT NULL,
    email             text NOT NULL,
    phone_no          text,
    password_hash     text NOT NULL,
    is_email_verified boolean DEFAULT false,
    is_admin          boolean DEFAULT false,
    is_active         boolean DEFAULT true,
    is_deleted        boolean DEFAULT false,
    created_at        timestamp with time zone,
    updated_at        timestamp with time zone,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

CREATE TABLE IF NOT EXISTS users_profile (
    id              uuid NOT NULL,
    user_id         text NOT NULL,
    first_name      text NOT NULL,
    last_name       text,
    bio             text,
    profile_image   text,
    native_language text NOT NULL,
    country_code    text,
    target_level    text,
    created_at      timestamp with time zone,
    updated_at      timestamp with time zone,
    CONSTRAINT users_profile_pkey PRIMARY KEY (id),
    CONSTRAINT chk_users_profile_target_level CHECK (target_level IN ('N1', 'N2', 'N3', 'N4', 'N5'))
);

CREATE TABLE IF NOT EXISTS users_settings (
    id                   uuid NOT NULL,
    user_id              text NOT NULL,
    current_time_zone    text,
    cursor_tail          boolean DEFAULT false,
    background_animation boolean DEFAULT false,
    daily_reminder       boolean DEFAULT false,
    reminder_time        text,
    created_at           timestamp with time zone,
    updated_at           timestamp with time zone,
    CONSTRAINT users_settings_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS user_verification (
    id           uuid NOT NULL,
    user_id      text NOT NULL,
    hash_code    text NOT NULL,
    expires_at   timestamp with time zone NOT NULL,
    verified_at  timestamp with time zone,
    last_sent_at timestamp with time zone,
    created_at   timestamp with time zone,
    updated_at   timestamp with time zone,
    CONSTRAINT user_verification_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         uuid NOT NULL,
    user_id    text NOT NULL,
    token_hash text NOT NULL,
    is_revoked boolean DEFAULT false,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id)
);
-- Older races left several rows for one user, which blocks the unique index
-- below. Keep each user's newest row; removing the rest only signs those
-- sessions out.
DELETE FROM refresh_tokens older
USING refresh_tokens newer
WHERE older.user_id = newer.user_id
  AND (older.updated_at, older.id) < (newer.updated_at, newer.id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);

CREATE TABLE IF NOT EXISTS password_resets (
    id                     uuid NOT NULL,
    user_id                text NOT NULL,
    code_hash              text NOT NULL,
    code_expires_at        timestamp with time zone NOT NULL,
    code_verified_at       timestamp with time zone,
    code_attempts          bigint DEFAULT 0 NOT NULL,
    code_last_sent_at      timestamp with time zone NOT NULL,
    code_window_started_at timestamp with time zone NOT NULL,
    code_sent_count        bigint DEFAULT 1 NOT NULL,
    reset_token_hash       text,
    reset_token_expires_at timestamp with time zone,
    reset_token_used_at    timestamp with time zone,
    created_at             timestamp with time zone,
    updated_at             timestamp with time zone,
    CONSTRAINT password_resets_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_password_resets_user_id ON password_resets (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_password_resets_token_hash ON password_resets (reset_token_hash);

CREATE TABLE IF NOT EXISTS oauth_identities (
    id               uuid NOT NULL,
    user_id          text NOT NULL,
    provider         text NOT NULL,
    provider_subject text NOT NULL,
    provider_email   text,
    email_verified   boolean DEFAULT false NOT NULL,
    last_used_at     timestamp with time zone NOT NULL,
    created_at       timestamp with time zone,
    updated_at       timestamp with time zone,
    CONSTRAINT oauth_identities_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_identities_provider_subject ON oauth_identities (provider, provider_subject);
CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_identities_user_provider ON oauth_identities (user_id, provider);

-- +goose Down
DROP TABLE IF EXISTS oauth_identities;
DROP TABLE IF EXISTS password_resets;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS user_verification;
DROP TABLE IF EXISTS users_settings;
DROP TABLE IF EXISTS users_profile;
DROP TABLE IF EXISTS users;

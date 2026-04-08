-- db/migrations/000001_init_table.up.sql

-- Create JLPT level enum type
CREATE TYPE jlpt_level AS ENUM ('N1', 'N2', 'N3', 'N4', 'N5');

CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- user info
    username            VARCHAR(255) NOT NULL,
    email               VARCHAR(255) UNIQUE NOT NULL,
    phone_no            VARCHAR(20) NULL,
    password_hash       VARCHAR(255) NOT NULL,

    -- user config
    is_email_verified   BOOL DEFAULT FALSE,
    is_admin            BOOL DEFAULT FALSE,
    is_active           BOOL DEFAULT TRUE,
    is_deleted          BOOL DEFAULT FALSE,

    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE users_profile (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- user details
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    first_name          VARCHAR(255) NOT NULL,
    last_name           VARCHAR(255) NULL,
    bio                 TEXT NULL,
    profile_image       VARCHAR(500) NULL,

    -- user config
    native_language     VARCHAR(10) NOT NULL,
    country_code        CHAR(2) NULL,
    target_level        jlpt_level NULL,

    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE users_settings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- location base config
    current_time_zone   VARCHAR(100) NULL,

    -- config
    cursor_tail         BOOL DEFAULT FALSE,
    background_animation    BOOL DEFAULT FALSE,
    daily_reminder      BOOL DEFAULT FALSE,
    reminder_time       TIME,

    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE user_verification (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- email verification 
    hash_code           VARCHAR(225) NOT NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    verified_at         TIMESTAMPTZ NULL,
    last_sent_at        TIMESTAMPTZ NULL,

    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uq_user_verification_user_id UNIQUE (user_id)
);

-- Index for fast user lookups
CREATE INDEX idx_users_profile_user_id ON users_profile(user_id);
CREATE INDEX idx_users_settings_user_id ON users_settings(user_id);
CREATE INDEX idx_user_verification_user_id ON user_verification(user_id);

-- Constraint 


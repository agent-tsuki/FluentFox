-- 000003_create_profiles.up.sql
-- Public-facing identity. Separated from users to keep the auth table lean.
-- One profile per user, created automatically on registration.

CREATE TABLE IF NOT EXISTS profiles (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID         NOT NULL,
    first_name      VARCHAR(100) NULL,
    last_name       VARCHAR(100) NULL,
    -- Plain text biography — no HTML stored
    bio             TEXT         NULL,
    -- Object storage key (not a full URL). Example: "profiles/abc-123/avatar.webp"
    profile_image   VARCHAR(500) NULL,
    -- BCP-47 language code. Example: 'en', 'id', 'th', 'zh'
    native_language VARCHAR(10)  NULL,
    -- ISO 3166-1 alpha-2. Example: 'JP', 'US', 'ID'
    country_code    CHAR(2)      NULL,
    -- JLPT level the user is working toward — personalises dashboard
    target_level    jlpt_level   NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_profiles_user_id  UNIQUE      (user_id),
    CONSTRAINT fk_profiles_users    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_profiles_user_id ON profiles (user_id);

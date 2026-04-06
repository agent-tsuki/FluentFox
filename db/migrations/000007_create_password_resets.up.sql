-- 000007_create_password_resets.up.sql
-- Hashed password reset tokens. Short-lived (15 minutes).
-- Old requests are not deleted — they simply expire and cannot be reused.

CREATE TABLE IF NOT EXISTS password_resets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL,
    -- SHA-256 hash of the token emailed to the user
    token_hash VARCHAR(64) NOT NULL,
    -- Set to NOW() + 15 minutes — very short window
    expires_at TIMESTAMPTZ NOT NULL,
    -- NULL = token available; set to NOW() when consumed
    used_at    TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_password_resets_users FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_password_resets_token_hash ON password_resets (token_hash);
CREATE INDEX idx_password_resets_user_id    ON password_resets (user_id);

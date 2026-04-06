-- 000006_create_email_verifications.up.sql
-- Hashed email verification tokens sent on registration and email change.
-- One row per pending verification.

CREATE TABLE IF NOT EXISTS email_verifications (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL,
    -- SHA-256 hash of the token emailed to the user
    token_hash VARCHAR(64) NOT NULL,
    -- Token expires 24 hours after creation
    expires_at TIMESTAMPTZ NOT NULL,
    -- NULL = not yet used; set to NOW() when user clicks the verification link
    used_at    TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_email_verifications_users FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_email_verifications_token_hash ON email_verifications (token_hash);
CREATE INDEX idx_email_verifications_user_id    ON email_verifications (user_id);

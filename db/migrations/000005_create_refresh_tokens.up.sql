-- 000005_create_refresh_tokens.up.sql
-- Hashed refresh tokens for JWT dual-token auth. One row per active session.
-- Revoked tokens are retained for audit — never deleted.

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID         NOT NULL,
    -- SHA-256 hash of the actual refresh token; raw token is sent to client only
    token_hash   VARCHAR(64)  NOT NULL,
    -- Browser/device string from User-Agent header — shown in "active sessions" list
    user_agent   VARCHAR(500) NULL,
    -- IPv4 or IPv6 address. VARCHAR(45) accommodates full IPv6 length.
    ip_address   VARCHAR(45)  NULL,
    -- Set to NOW() + JWT_REFRESH_EXPIRY_DAYS on creation
    expires_at   TIMESTAMPTZ  NOT NULL,
    -- NULL = still valid; set to NOW() on logout or token rotation
    revoked_at   TIMESTAMPTZ  NULL,
    -- Updated every time this token is used to refresh
    last_used_at TIMESTAMPTZ  NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_refresh_tokens_token_hash UNIQUE      (token_hash),
    CONSTRAINT fk_refresh_tokens_users      FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- Used on every token validation (hash lookup)
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);
-- Used to list or revoke all sessions for a user
CREATE INDEX idx_refresh_tokens_user_id    ON refresh_tokens (user_id);
-- Used by cleanup job to purge expired tokens
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);

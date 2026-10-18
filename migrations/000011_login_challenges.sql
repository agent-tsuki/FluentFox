-- +goose Up
-- A password login from a VPN must also prove access to the account's email.
-- Each row is one emailed code; only its HMAC is stored.
CREATE TABLE login_challenges (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash   text NOT NULL,
    attempts    integer NOT NULL DEFAULT 0,
    location    jsonb,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- Serves the per-user cooldown and daily limit lookups.
CREATE INDEX idx_login_challenges_user_created ON login_challenges (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS login_challenges;

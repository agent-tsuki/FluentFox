-- +goose Up
-- An emailed invite that hands its grants to whoever accepts it with the same
-- verified email. Only the SHA-256 of the link token is stored.
CREATE TABLE staff_invites (
    id          uuid PRIMARY KEY,
    email       text NOT NULL,
    token_hash  bytea NOT NULL UNIQUE,
    invited_by  uuid NOT NULL,
    grants      jsonb NOT NULL DEFAULT '[]',
    expires_at  timestamptz NOT NULL,
    accepted_at timestamptz,
    accepted_by uuid,
    revoked_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_staff_invites_created ON staff_invites (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS staff_invites;

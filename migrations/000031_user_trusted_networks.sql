-- +goose Up
-- Networks where a user has entered a VPN login code. A password login from
-- one of them within the trust window skips the code. Only the masked prefix
-- (/24 IPv4, /48 IPv6) is stored, never the full address.
CREATE TABLE user_trusted_networks (
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    ip_prefix   text NOT NULL,
    verified_at timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, ip_prefix),
    CONSTRAINT chk_user_trusted_networks_prefix_len CHECK (char_length(ip_prefix) BETWEEN 1 AND 50)
);

-- +goose Down
DROP TABLE IF EXISTS user_trusted_networks;

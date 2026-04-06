-- 000022_create_srs_deck_settings.up.sql
-- Per-user per-card-type SRS algorithm configuration.
-- One row per user per card_type (4 possible types × N users).

CREATE TABLE IF NOT EXISTS srs_deck_settings (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID          NOT NULL,
    card_type            srs_card_type NOT NULL,
    -- Max new cards introduced per calendar day for this type
    new_cards_per_day    INTEGER       NOT NULL DEFAULT 20,
    -- Cap on reviews per day to prevent overwhelming the user
    review_limit_per_day INTEGER       NOT NULL DEFAULT 100,
    -- Lapses needed to auto-flag a card as is_leech = TRUE
    leech_threshold      INTEGER       NOT NULL DEFAULT 8,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_srs_deck_settings        UNIQUE (user_id, card_type),
    CONSTRAINT chk_srs_deck_new_cards      CHECK  (new_cards_per_day BETWEEN 1 AND 9999),
    CONSTRAINT chk_srs_deck_leech          CHECK  (leech_threshold BETWEEN 3 AND 20),
    CONSTRAINT fk_srs_deck_settings_users  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_srs_deck_settings_user ON srs_deck_settings (user_id);

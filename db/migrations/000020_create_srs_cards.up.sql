-- 000020_create_srs_cards.up.sql
-- Core SRS entity. One row per user per content item per card face.
-- Every column maps directly to a field in the fsrs.Card Go struct (go-fsrs/v3).
--
-- content_id is a polymorphic FK — which table it points to depends on card_type:
--   card_type='vocabulary' → vocabulary.id
--   card_type='kanji'      → kanji_entries.id
--   card_type='character'  → characters.id
--   card_type='concept'    → concepts.id
-- FK integrity is enforced at application layer (PostgreSQL cannot do polymorphic FK).

CREATE TABLE IF NOT EXISTS srs_cards (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID           NOT NULL,
    card_type        srs_card_type  NOT NULL,
    -- Polymorphic reference — points to UUID PK in content tables
    content_id       UUID           NOT NULL,
    card_face        srs_card_face  NOT NULL,

    -- FSRS algorithm state (mirrors fsrs.Card struct)
    stability        FLOAT          NOT NULL DEFAULT 0,
    difficulty       FLOAT          NOT NULL DEFAULT 0,
    elapsed_days     INTEGER        NOT NULL DEFAULT 0,
    scheduled_days   INTEGER        NOT NULL DEFAULT 0,
    reps             INTEGER        NOT NULL DEFAULT 0,
    lapses           INTEGER        NOT NULL DEFAULT 0,
    card_state       srs_card_state NOT NULL DEFAULT 'New',

    -- When this card is due. Query: WHERE user_id=? AND next_review_at <= NOW()
    next_review_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    -- NULL if card has never been reviewed (state = 'New')
    last_reviewed_at TIMESTAMPTZ    NULL,

    -- Application flags (not part of FSRS algorithm)
    is_suspended     BOOLEAN        NOT NULL DEFAULT FALSE,
    -- Auto-set TRUE when lapses >= leech_threshold from srs_deck_settings
    is_leech         BOOLEAN        NOT NULL DEFAULT FALSE,

    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    -- One scheduling state per user per content item per direction
    CONSTRAINT uq_srs_cards         UNIQUE (user_id, card_type, content_id, card_face),
    CONSTRAINT chk_srs_difficulty   CHECK  (difficulty >= 0 AND difficulty <= 10),
    CONSTRAINT chk_srs_reps         CHECK  (reps >= 0),
    CONSTRAINT chk_srs_lapses       CHECK  (lapses >= 0),
    CONSTRAINT fk_srs_cards_users   FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- MOST CRITICAL INDEX: used for every review session query ("all cards due for this user")
CREATE INDEX idx_srs_cards_due   ON srs_cards (user_id, next_review_at);
-- Filter by type for per-deck views
CREATE INDEX idx_srs_cards_type  ON srs_cards (user_id, card_type);
-- Dashboard: how many cards in each state
CREATE INDEX idx_srs_cards_state ON srs_cards (user_id, card_state);
-- Dashboard: show weak cards list
CREATE INDEX idx_srs_cards_leech ON srs_cards (user_id, is_leech);

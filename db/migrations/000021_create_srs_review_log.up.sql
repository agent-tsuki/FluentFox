-- 000021_create_srs_review_log.up.sql
-- Append-only audit log of every review event. One row per review — never updated.
-- Powers review heatmap, retention rate, and weak card analytics on the dashboard.
-- Maps to the fsrs.ReviewLog Go struct.

CREATE TABLE IF NOT EXISTS srs_review_log (
    id             UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id        UUID           NOT NULL,
    -- Denormalised from srs_cards for fast user-level queries without join
    user_id        UUID           NOT NULL,
    -- FSRS rating: 1=Again, 2=Hard, 3=Good, 4=Easy
    rating         INTEGER        NOT NULL,
    -- Card state before this review — used to detect graduation (Learning → Review)
    state_before   srs_card_state NOT NULL,
    -- Card state after this review
    state_after    srs_card_state NOT NULL,
    -- How many days until next review (after this review)
    scheduled_days INTEGER        NOT NULL,
    -- How many days since the last review
    elapsed_days   INTEGER        NOT NULL,
    reviewed_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_review_log_rating  CHECK       (rating BETWEEN 1 AND 4),
    CONSTRAINT fk_review_log_cards    FOREIGN KEY (card_id) REFERENCES srs_cards (id) ON DELETE CASCADE,
    CONSTRAINT fk_review_log_users    FOREIGN KEY (user_id) REFERENCES users     (id) ON DELETE CASCADE
);

-- Review history for a single card
CREATE INDEX idx_review_log_card_id     ON srs_review_log (card_id, reviewed_at);
-- Daily review count, heatmap, and streak validation
CREATE INDEX idx_review_log_user_date   ON srs_review_log (user_id, reviewed_at);
-- Retention rate calculation (rating >= 3 = correct)
CREATE INDEX idx_review_log_user_rating ON srs_review_log (user_id, rating);

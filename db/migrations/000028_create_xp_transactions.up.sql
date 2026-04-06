-- 000028_create_xp_transactions.up.sql
-- Append-only XP ledger. One row per XP credit or debit event.
-- Provides a full audit trail; user_xp.total_xp and xp_spent are derived from this table.
-- append-only log: created_at only, no updated_at.

CREATE TABLE IF NOT EXISTS xp_transactions (
    id               UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID                 NOT NULL,
    -- Credit vs debit direction
    type             xp_transaction_type  NOT NULL,
    -- What triggered this XP event
    source_type      xp_source_type       NOT NULL,
    -- XP amount. Positive for earned/refunded, negative for spent.
    amount           INTEGER              NOT NULL,
    -- Optional references to the context that generated the XP
    chapter_id       INTEGER              NULL,
    quiz_session_id  UUID                 NULL,
    created_at       TIMESTAMPTZ          NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_xp_transactions_users         FOREIGN KEY (user_id)         REFERENCES users         (id) ON DELETE CASCADE,
    CONSTRAINT fk_xp_transactions_chapters      FOREIGN KEY (chapter_id)      REFERENCES chapters      (id) ON DELETE SET NULL,
    CONSTRAINT fk_xp_transactions_quiz_sessions FOREIGN KEY (quiz_session_id) REFERENCES quiz_sessions (id) ON DELETE SET NULL
);

CREATE INDEX idx_xp_transactions_user_id ON xp_transactions (user_id, created_at);
CREATE INDEX idx_xp_transactions_source  ON xp_transactions (user_id, source_type);

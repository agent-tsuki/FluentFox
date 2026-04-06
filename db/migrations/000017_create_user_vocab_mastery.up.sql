-- 000017_create_user_vocab_mastery.up.sql
-- Tracks which vocabulary words a user has manually marked as mastered.
-- Replaces the localStorage implementation (masteryKey pattern) in the frontend.
-- One row per user per vocab word, created on first toggle.

CREATE TABLE IF NOT EXISTS user_vocab_mastery (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL,
    vocab_id    INTEGER     NOT NULL,
    is_mastered BOOLEAN     NOT NULL DEFAULT FALSE,
    -- Set to NOW() when is_mastered flips to TRUE; reset to NULL when un-mastered
    mastered_at TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_vocab_mastery  UNIQUE      (user_id, vocab_id),
    CONSTRAINT fk_uvm_users           FOREIGN KEY (user_id)  REFERENCES users      (id) ON DELETE CASCADE,
    CONSTRAINT fk_uvm_vocabulary      FOREIGN KEY (vocab_id) REFERENCES vocabulary (id) ON DELETE CASCADE
);

CREATE INDEX idx_uvm_user_id       ON user_vocab_mastery (user_id);
-- Used for mastery count queries on dashboard
CREATE INDEX idx_uvm_user_mastered ON user_vocab_mastery (user_id, is_mastered);
CREATE INDEX idx_uvm_vocab_id      ON user_vocab_mastery (vocab_id);

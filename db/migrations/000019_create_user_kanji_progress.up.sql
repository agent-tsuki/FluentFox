-- 000019_create_user_kanji_progress.up.sql
-- Tracks a user's progress through kanji at each JLPT level.
-- One row per user per level (not per individual kanji entry).

CREATE TABLE IF NOT EXISTS user_kanji_progress (
    id           UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID            NOT NULL,
    jlpt_level   jlpt_level      NOT NULL,
    status       progress_status NOT NULL DEFAULT 'not_started',
    -- 0–100. Most recent kanji quiz score for this level. NULL until first quiz.
    score_pct    INTEGER         NULL,
    -- Set when status transitions to 'completed'
    completed_at TIMESTAMPTZ     NULL,
    created_at   TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_kanji_progress       UNIQUE      (user_id, jlpt_level),
    CONSTRAINT chk_ukp_score                CHECK       (score_pct IS NULL OR (score_pct >= 0 AND score_pct <= 100)),
    CONSTRAINT fk_user_kanji_progress_users FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_kanji_progress_user_id ON user_kanji_progress (user_id);

-- 000018_create_user_character_progress.up.sql
-- Tracks a user's progress through hiragana or katakana as a whole script.
-- One row per user per script (not per individual character).

CREATE TABLE IF NOT EXISTS user_character_progress (
    id           UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID             NOT NULL,
    script       character_script NOT NULL,
    status       progress_status  NOT NULL DEFAULT 'not_started',
    -- 0–100. Most recent quiz score percentage for this script. NULL until first quiz.
    score_pct    INTEGER          NULL,
    -- Set when status transitions to 'completed'
    completed_at TIMESTAMPTZ      NULL,
    created_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_character_progress       UNIQUE      (user_id, script),
    CONSTRAINT chk_ucp_score                    CHECK       (score_pct IS NULL OR (score_pct >= 0 AND score_pct <= 100)),
    CONSTRAINT fk_user_character_progress_users FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_char_progress_user_id ON user_character_progress (user_id);

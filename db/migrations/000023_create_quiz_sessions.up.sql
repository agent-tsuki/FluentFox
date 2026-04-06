-- 000023_create_quiz_sessions.up.sql
-- Records a completed quiz attempt. One row per quiz taken.
-- Linked optionally to a chapter (grammar/vocab quiz), a JLPT level (kanji quiz),
-- or standalone (character quiz).

CREATE TABLE IF NOT EXISTS quiz_sessions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL,
    quiz_type        quiz_type   NOT NULL,

    -- Context: at most one of these is set
    -- Set for grammar and vocabulary quizzes
    chapter_id       UUID        NULL,
    -- Set for kanji quizzes
    jlpt_level       jlpt_level  NULL,

    -- Results
    total_questions  INTEGER     NOT NULL,
    correct_answers  INTEGER     NOT NULL DEFAULT 0,
    -- (correct_answers / total_questions) * 100, rounded
    score_pct        INTEGER     NOT NULL DEFAULT 0,

    -- Timing
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- NULL = quiz started but not yet submitted
    completed_at     TIMESTAMPTZ NULL,

    CONSTRAINT chk_quiz_score    CHECK       (score_pct >= 0 AND score_pct <= 100),
    CONSTRAINT chk_quiz_totals   CHECK       (correct_answers <= total_questions),
    CONSTRAINT fk_quiz_sessions_users    FOREIGN KEY (user_id)    REFERENCES users    (id) ON DELETE CASCADE,
    CONSTRAINT fk_quiz_sessions_chapters FOREIGN KEY (chapter_id) REFERENCES chapters (id) ON DELETE SET NULL
);

CREATE INDEX idx_quiz_sessions_user_id   ON quiz_sessions (user_id);
CREATE INDEX idx_quiz_sessions_user_type ON quiz_sessions (user_id, quiz_type);
CREATE INDEX idx_quiz_sessions_chapter   ON quiz_sessions (chapter_id);

-- 000024_create_quiz_answers.up.sql
-- One row per question within a quiz session.
-- Records exactly what was asked and what the user answered.
-- The question source is identified by which FK column is non-null.

CREATE TABLE IF NOT EXISTS quiz_answers (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     UUID         NOT NULL,

    -- Content reference — exactly one of these is non-null
    vocab_id       INTEGER      NULL,
    character_id   INTEGER      NULL,
    kanji_id       INTEGER      NULL,
    concept_id     INTEGER      NULL,

    -- Question data stored for historical accuracy (content may change over time)
    question_text  TEXT         NOT NULL,
    correct_answer VARCHAR(500) NOT NULL,
    user_answer    VARCHAR(500) NOT NULL,
    is_correct     BOOLEAN      NOT NULL,
    answered_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_quiz_answers_sessions   FOREIGN KEY (session_id)   REFERENCES quiz_sessions (id) ON DELETE CASCADE,
    CONSTRAINT fk_quiz_answers_vocab      FOREIGN KEY (vocab_id)     REFERENCES vocabulary    (id) ON DELETE SET NULL,
    CONSTRAINT fk_quiz_answers_characters FOREIGN KEY (character_id) REFERENCES characters    (id) ON DELETE SET NULL,
    CONSTRAINT fk_quiz_answers_kanji      FOREIGN KEY (kanji_id)     REFERENCES kanji_entries (id) ON DELETE SET NULL,
    CONSTRAINT fk_quiz_answers_concepts   FOREIGN KEY (concept_id)   REFERENCES concepts      (id) ON DELETE SET NULL
);

CREATE INDEX idx_quiz_answers_session_id ON quiz_answers (session_id);
-- Per-item analysis: which questions a user consistently gets wrong
CREATE INDEX idx_quiz_answers_correct    ON quiz_answers (session_id, is_correct);

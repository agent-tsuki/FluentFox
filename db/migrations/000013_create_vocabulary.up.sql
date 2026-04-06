-- 000013_create_vocabulary.up.sql
-- Chapter vocabulary words shown in the sidebar of a chapter.
-- Sourced from the Vocabulary Grid table in each MDX file.
-- Also the content source for vocabulary SRS cards.

CREATE TABLE IF NOT EXISTS vocabulary (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id  UUID         NOT NULL,
    -- Kanji or kana form. Example: "銀行員"
    word        VARCHAR(100) NOT NULL,
    -- Hiragana reading. Example: "ぎんこういん"
    reading     VARCHAR(100) NOT NULL,
    -- English meaning. Example: "Bank employee"
    meaning     VARCHAR(255) NOT NULL,
    jlpt_level  jlpt_level   NOT NULL,
    -- Position in the vocabulary list for this chapter
    order_index INTEGER      NOT NULL,
    -- Object storage key for pronunciation audio. NULL = no audio available.
    audio_key   VARCHAR(500) NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_vocabulary_chapters FOREIGN KEY (chapter_id) REFERENCES chapters (id) ON DELETE CASCADE
);

CREATE INDEX idx_vocabulary_chapter_id    ON vocabulary (chapter_id);
CREATE INDEX idx_vocabulary_chapter_order ON vocabulary (chapter_id, order_index);
CREATE INDEX idx_vocabulary_jlpt_level    ON vocabulary (jlpt_level);

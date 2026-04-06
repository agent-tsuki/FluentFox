-- 000015_create_kanji_entries.up.sql
-- JLPT kanji list. ~2,136 entries across N5–N1.
-- Seeded from db/seeds/kanji.sql — never updated by users.

CREATE TABLE IF NOT EXISTS kanji_entries (
    id           SERIAL       PRIMARY KEY,
    -- The kanji character. Example: "私", "学", "人"
    character    VARCHAR(5)   NOT NULL,
    -- On-reading (Chinese-derived), stored in katakana. Comma-separated. Example: "ガク, ハク"
    onyomi       VARCHAR(255) NULL,
    -- Kun-reading (native Japanese), stored in hiragana. Dot notation for okurigana. Example: "まな.ぶ"
    kunyomi      VARCHAR(255) NULL,
    -- Primary English meaning(s), comma-separated. Example: "I, me, my"
    meaning      VARCHAR(500) NOT NULL,
    -- Number of strokes to write the character. NULL if not available.
    stroke_count INTEGER      NULL,
    jlpt_level   jlpt_level   NOT NULL,
    -- Study order within the JLPT level
    order_index  INTEGER      NOT NULL,
    -- Object storage key for pronunciation audio
    audio_key    VARCHAR(500) NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_kanji_entries_character UNIQUE (character)
);

CREATE INDEX idx_kanji_level       ON kanji_entries (jlpt_level);
CREATE INDEX idx_kanji_level_order ON kanji_entries (jlpt_level, order_index);

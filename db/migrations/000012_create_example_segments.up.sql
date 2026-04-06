-- 000012_create_example_segments.up.sql
-- Stores the Japanese sentence of an example as an ordered list of typed segments.
-- DB representation of: JapaneseSegment = string | {kanji, reading, meaning}
-- Each row is one token in the sentence.

CREATE TABLE IF NOT EXISTS example_segments (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    example_id  UUID         NOT NULL,
    -- Position in the sentence, starts at 0
    order_index INTEGER      NOT NULL,
    type        segment_type NOT NULL,

    -- Populated when type = 'plain_text'
    -- Raw Japanese text, may contain punctuation and particles
    text_content VARCHAR(500) NULL,

    -- Populated when type = 'interactive_word'
    kanji   VARCHAR(100) NULL,  -- kanji or kana form, e.g. "私"
    reading VARCHAR(100) NULL,  -- hiragana reading, e.g. "わたし"
    meaning VARCHAR(255) NULL,  -- English meaning, e.g. "I / Me"

    -- plain_text segments must have text_content
    CONSTRAINT chk_segments_plain_text  CHECK (type != 'plain_text'      OR text_content IS NOT NULL),
    -- interactive_word segments must have all three annotation fields
    CONSTRAINT chk_segments_interactive CHECK (type != 'interactive_word' OR
                                               (kanji IS NOT NULL AND reading IS NOT NULL AND meaning IS NOT NULL)),

    CONSTRAINT fk_example_segments_examples FOREIGN KEY (example_id) REFERENCES concept_examples (id) ON DELETE CASCADE
);

CREATE INDEX idx_example_segments_example_id ON example_segments (example_id);
CREATE INDEX idx_example_segments_order      ON example_segments (example_id, order_index);

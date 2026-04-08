-- db/migrations/000002_create_vocab_grammar.up.sql

-- Create lama enum type
CREATE TYPE kana_type AS ENUM ('hiragana', 'katakana');

-- Tables
CREATE TABLE kanji (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    word        VARCHAR NOT NULL,
    onyomi      VARCHAR NULL,
    kunyomi     VARCHAR NULL,
    meaning     VARCHAR NOT NULL,
    hiragana    VARCHAR NULL,
    romaji      VARCHAR NULL,
    target_level  jlpt_level NOT NULL,
    image_key   VARCHAR(500) NULL,
    audio_key   VARCHAR(500) NULL,

    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE kanas (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    character   VARCHAR NOT NULL,
    romanji     VARCHAR NOT NULL,
    kana_type   kana_type NOT NULL,
    target_level  jlpt_level NOT NULL,
    stroke_order INTEGER NULL,  

    image_key   VARCHAR(500) NULL,
    audio_key   VARCHAR(500) NULL,

    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE vocabulary (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kanji_id    UUID NULL REFERENCES kanji(id),

    word        VARCHAR NOT NULL,
    meaning     VARCHAR NOT NULL,
    hiragana    VARCHAR NULL,
    romaji      VARCHAR NULL,
    target_level  jlpt_level NOT NULL,

    image_key   VARCHAR(500) NULL,
    audio_key   VARCHAR(500) NULL,

    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Many-to-Many relation for kanji and vocab
CREATE TABLE kanji_vocabulary (
    kanji_id        UUID NOT NULL REFERENCES kanji(id) ON DELETE CASCADE,
    vocabulary_id   UUID NOT NULL REFERENCES vocabulary(id) ON DELETE CASCADE,
    PRIMARY KEY (kanji_id, vocabulary_id)
);

-- Indexes
CREATE INDEX idx_vocabulary_kanji_id ON vocabulary(kanji_id);

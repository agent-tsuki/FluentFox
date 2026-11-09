-- +goose Up
-- Grammar chapters as a learning area: each chapter has words (the vocab_n5
-- track) and practice sentences (the grammar_n5 track). Both tracks use the
-- chapter rows as their units, under their own unit type, so a chapter keeps
-- a vocab quiz score and an exercise score side by side in unit_progress.
--
-- Enum values added here are only used by later transactions (PostgreSQL 12+
-- allows ADD VALUE inside a transaction on that condition).
ALTER TYPE learning_track ADD VALUE 'vocab_n5';
ALTER TYPE learning_track ADD VALUE 'grammar_n5';
ALTER TYPE srs_item_type ADD VALUE 'vocab';
ALTER TYPE srs_item_type ADD VALUE 'grammar_sentence';
ALTER TYPE unit_type ADD VALUE 'vocab_chapter';
ALTER TYPE unit_type ADD VALUE 'grammar_chapter';
ALTER TYPE question_type ADD VALUE 'vocab_meaning_mcq';
ALTER TYPE question_type ADD VALUE 'vocab_meaning_typed';
ALTER TYPE question_type ADD VALUE 'grammar_gap_mcq';
ALTER TYPE question_type ADD VALUE 'grammar_gap_typed';
ALTER TYPE question_type ADD VALUE 'grammar_translate_mcq';

CREATE TYPE chapter_completion AS ENUM ('auto', 'manual');

CREATE TABLE grammar_chapter (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    level      jlpt_level NOT NULL,
    slug       text NOT NULL,
    number     int NOT NULL,
    title      text NOT NULL,
    position   int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_grammar_chapter_level_slug UNIQUE (level, slug)
);

CREATE TABLE chapter_word (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id uuid NOT NULL REFERENCES grammar_chapter (id) ON DELETE CASCADE,
    position   int NOT NULL,
    word       text NOT NULL,
    kana       text NOT NULL,
    romaji     text NOT NULL,
    meaning    text NOT NULL,
    hint       text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_chapter_word_chapter_word UNIQUE (chapter_id, word)
);

CREATE TABLE grammar_sentence (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id  uuid NOT NULL REFERENCES grammar_chapter (id) ON DELETE CASCADE,
    position    int NOT NULL,
    -- The sentence with {} where the answer goes.
    sentence    text NOT NULL,
    answer      text NOT NULL,
    distractors text[] NOT NULL,
    also_accept text[] NOT NULL DEFAULT '{}',
    -- Typed romaji that kana conversion cannot reach, e.g. "wa" for は.
    romaji      text[] NOT NULL DEFAULT '{}',
    translation text NOT NULL,
    explanation text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_grammar_sentence_chapter_sentence UNIQUE (chapter_id, sentence),
    CONSTRAINT chk_grammar_sentence_gap CHECK (position('{}' IN sentence) > 0)
);

ALTER TABLE learning_settings
    ADD COLUMN chapter_pass_pct smallint NOT NULL DEFAULT 70,
    ADD COLUMN chapter_completion chapter_completion NOT NULL DEFAULT 'auto',
    ADD CONSTRAINT chk_learning_settings_chapter_pass_pct CHECK (chapter_pass_pct BETWEEN 50 AND 100);

-- +goose Down
-- Enum values cannot be dropped in PostgreSQL; the added ones stay unused.
ALTER TABLE learning_settings
    DROP CONSTRAINT chk_learning_settings_chapter_pass_pct,
    DROP COLUMN chapter_completion,
    DROP COLUMN chapter_pass_pct;
DROP TABLE IF EXISTS grammar_sentence;
DROP TABLE IF EXISTS chapter_word;
DROP TABLE IF EXISTS grammar_chapter;
DROP TYPE IF EXISTS chapter_completion;

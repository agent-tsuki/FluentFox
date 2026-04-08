-- db/migrations/000003_update_vocab_create_grammar.up.sql

ALTER TABLE vocabulary DROP COLUMN kanji_id;

CREATE TABLE grammar (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_no   INTEGER NOT NULL,
    title        VARCHAR NOT NULL,
    target_level jlpt_level NOT NULL,
    content      TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

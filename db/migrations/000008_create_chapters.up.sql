-- 000008_create_chapters.up.sql
-- Top-level unit of grammar instruction.
-- One chapter = one MDX file = one lesson page in the UI.
-- Uses SERIAL PK: internal content table, never exposed as a resource identifier.

CREATE TABLE IF NOT EXISTS chapters (
    id           SERIAL       PRIMARY KEY,
    level        jlpt_level   NOT NULL,
    -- Sequential number within the level, starts at 1. Used in URL: /grammar/N5/1
    chapter_no   INTEGER      NOT NULL,
    title        VARCHAR(255) NOT NULL,
    -- One to two sentence summary shown in chapter list and at the top of the chapter page
    description  TEXT         NOT NULL,
    -- Absolute ordering across all chapters within a level; determines prev/next navigation
    order_index  INTEGER      NOT NULL,
    -- FALSE = hidden from students; allows admin to draft/preview before publishing
    is_published BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- No two chapters in the same level can share a number
    CONSTRAINT uq_chapters_level_chapter_no UNIQUE (level, chapter_no)
);

CREATE INDEX idx_chapters_level        ON chapters (level);
-- Used for chapter list query ordered by level + position
CREATE INDEX idx_chapters_level_order  ON chapters (level, order_index);
CREATE INDEX idx_chapters_is_published ON chapters (is_published);

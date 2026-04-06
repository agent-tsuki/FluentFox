-- 000008_create_chapters.up.sql
-- Top-level unit of grammar instruction.
-- One chapter = one MDX file = one lesson page in the UI.

CREATE TABLE IF NOT EXISTS chapters (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        VARCHAR(255) NOT NULL,
    jlpt_level  jlpt_level   NOT NULL,
    title       VARCHAR(255) NOT NULL,
    -- One to two sentence summary shown in chapter list and at the top of the chapter page
    description TEXT         NOT NULL,
    -- Absolute ordering across all chapters; determines prev/next navigation
    order_index INTEGER      NOT NULL,
    -- FALSE = hidden from students; allows admin to draft/preview before publishing
    published   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_chapters_slug UNIQUE (slug)
);

CREATE INDEX idx_chapters_jlpt_level ON chapters (jlpt_level);
-- Used for chapter list query ordered by level + position
CREATE INDEX idx_chapters_order      ON chapters (jlpt_level, order_index);
CREATE INDEX idx_chapters_published  ON chapters (published);

-- 000010_create_concepts.up.sql
-- A grammar concept within a chapter. Each chapter has one to many concepts.
-- Each ## Concept XX heading in the MDX file produces one row.

CREATE TABLE IF NOT EXISTS concepts (
    id          SERIAL       PRIMARY KEY,
    chapter_id  INTEGER      NOT NULL,
    title       VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL,
    -- Core rule distilled to one sentence; parsed from "> **Key Rule:**" blockquotes
    key_rule    TEXT         NULL,
    -- Supplementary tip; parsed from "> **Pro Tip:**" or "> **Note:**" blockquotes
    note        TEXT         NULL,
    -- Position of this concept within its chapter
    order_index INTEGER      NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_concepts_chapters FOREIGN KEY (chapter_id) REFERENCES chapters (id) ON DELETE CASCADE
);

CREATE INDEX idx_concepts_chapter_id    ON concepts (chapter_id);
CREATE INDEX idx_concepts_chapter_order ON concepts (chapter_id, order_index);

-- 000009_create_cultural_insights.up.sql
-- Cultural note shown in the sidebar of a chapter.
-- One insight per chapter maximum. Optional — not every chapter needs one.

CREATE TABLE IF NOT EXISTS cultural_insights (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id UUID        NOT NULL,
    title      VARCHAR(255) NOT NULL,
    -- HTML content; sanitised by sync-content tool before insertion
    body       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Maximum one insight per chapter
    CONSTRAINT uq_cultural_insights_chapter_id UNIQUE      (chapter_id),
    CONSTRAINT fk_cultural_insights_chapters   FOREIGN KEY (chapter_id) REFERENCES chapters (id) ON DELETE CASCADE
);

CREATE INDEX idx_cultural_insights_chapter_id ON cultural_insights (chapter_id);

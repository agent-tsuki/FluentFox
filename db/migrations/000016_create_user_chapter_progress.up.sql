-- 000016_create_user_chapter_progress.up.sql
-- Tracks whether a user has started or completed each chapter.
-- One row per user per chapter, created lazily on first chapter visit.

CREATE TABLE IF NOT EXISTS user_chapter_progress (
    id           UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID            NOT NULL,
    chapter_id   INTEGER         NOT NULL,
    status       progress_status NOT NULL DEFAULT 'not_started',
    -- Set when user clicks "Mark Chapter Complete"
    completed_at TIMESTAMPTZ     NULL,
    created_at   TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_chapter_progress UNIQUE      (user_id, chapter_id),
    CONSTRAINT fk_ucp_users             FOREIGN KEY (user_id)    REFERENCES users    (id) ON DELETE CASCADE,
    -- RESTRICT: cannot delete a chapter that users have progress on
    CONSTRAINT fk_ucp_chapters          FOREIGN KEY (chapter_id) REFERENCES chapters (id) ON DELETE RESTRICT
);

CREATE INDEX idx_ucp_user_id     ON user_chapter_progress (user_id);
CREATE INDEX idx_ucp_user_status ON user_chapter_progress (user_id, status);
CREATE INDEX idx_ucp_chapter_id  ON user_chapter_progress (chapter_id);

-- 000026_create_streak_activity_log.up.sql
-- Append-only log of qualifying activity per calendar day per user.
-- One row per user per day — a second activity on the same day is a no-op.
-- Used by streak calculation to determine if a day was active.
-- append-only log: created_at only, no updated_at.

CREATE TABLE IF NOT EXISTS streak_activity_log (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL,
    -- Calendar date in the user's local timezone (from user_settings.timezone)
    activity_date DATE        NOT NULL,
    -- Free-form activity type string, e.g. 'srs_review', 'quiz', 'chapter_read'
    activity_type TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One activity record per user per day
    CONSTRAINT uq_streak_activity_log_user_date UNIQUE      (user_id, activity_date),
    CONSTRAINT fk_streak_activity_log_users     FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_streak_activity_log_user_date ON streak_activity_log (user_id, activity_date);

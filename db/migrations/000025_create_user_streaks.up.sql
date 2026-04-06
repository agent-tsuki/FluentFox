-- 000025_create_user_streaks.up.sql
-- Per-user streak counters. One row per user.
-- streak_freezes_available is incremented when user purchases a streak_freeze shop item.

CREATE TABLE IF NOT EXISTS user_streaks (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                  UUID        NOT NULL,
    current_streak           INTEGER     NOT NULL DEFAULT 0,
    longest_streak           INTEGER     NOT NULL DEFAULT 0,
    -- The calendar date (in the user's timezone) of their last qualifying activity
    last_activity_date       DATE        NULL,
    -- Streak freezes granted by shop purchases; decremented when a freeze is consumed
    streak_freezes_available INTEGER     NOT NULL DEFAULT 0,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_streaks_user_id  UNIQUE      (user_id),
    CONSTRAINT chk_user_streaks_current CHECK       (current_streak >= 0),
    CONSTRAINT chk_user_streaks_longest CHECK       (longest_streak >= 0),
    CONSTRAINT chk_user_streaks_freezes CHECK       (streak_freezes_available >= 0),
    CONSTRAINT fk_user_streaks_users    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_streaks_user_id ON user_streaks (user_id);

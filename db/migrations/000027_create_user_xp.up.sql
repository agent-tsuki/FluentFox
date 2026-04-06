-- 000027_create_user_xp.up.sql
-- Running XP totals and computed level for each user. One row per user.
-- xp_spent tracks XP consumed by shop purchases (for balance checks).

CREATE TABLE IF NOT EXISTS user_xp (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL,
    -- Lifetime XP earned (never decremented)
    total_xp   INTEGER     NOT NULL DEFAULT 0,
    -- Total XP spent in the shop (used to compute available balance: total_xp - xp_spent)
    xp_spent   INTEGER     NOT NULL DEFAULT 0,
    -- Computed level, updated whenever total_xp crosses an xp_level_config threshold
    level      INTEGER     NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_xp_user_id  UNIQUE (user_id),
    CONSTRAINT chk_user_xp_total   CHECK  (total_xp >= 0),
    CONSTRAINT chk_user_xp_spent   CHECK  (xp_spent >= 0),
    CONSTRAINT chk_user_xp_level   CHECK  (level >= 1),
    CONSTRAINT fk_user_xp_users    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- Leaderboard ordering
CREATE INDEX idx_user_xp_user_id ON user_xp (user_id);
CREATE INDEX idx_user_xp_total   ON user_xp (total_xp DESC);

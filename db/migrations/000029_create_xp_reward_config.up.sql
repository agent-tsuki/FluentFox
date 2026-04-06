-- 000029_create_xp_reward_config.up.sql
-- Lookup table: base XP amount for each source_type event.
-- Lookup/config table — uses SERIAL PK (integer), never exposed as a resource identifier.
-- Seeded with defaults; admin can update base_xp values at runtime.

CREATE TABLE IF NOT EXISTS xp_reward_config (
    id          SERIAL          PRIMARY KEY,
    source_type xp_source_type  NOT NULL,
    base_xp     INTEGER         NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_xp_reward_config_source UNIQUE (source_type),
    CONSTRAINT chk_xp_reward_config_base  CHECK  (base_xp >= 0)
);

-- Seed default XP reward values
INSERT INTO xp_reward_config (source_type, base_xp) VALUES
    ('chapter_completed',   50),
    ('vocab_mastered',       5),
    ('quiz_perfect_score',  30),
    ('quiz_completed',      10),
    ('streak_milestone',    50),
    ('kanji_level_cleared', 100),
    ('character_cleared',   75),
    ('srs_card_graduated',   2),
    ('daily_login',          5),
    ('admin_granted',        0)
ON CONFLICT (source_type) DO NOTHING;

-- 000030_create_xp_level_config.up.sql
-- XP thresholds required to reach each level.
-- Lookup/config table — integer PK (level number IS the identifier).

CREATE TABLE IF NOT EXISTS xp_level_config (
    level        INTEGER     PRIMARY KEY,
    -- Total XP required to reach this level
    xp_required  INTEGER     NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_xp_level_config_level       CHECK (level >= 1),
    CONSTRAINT chk_xp_level_config_xp_required CHECK (xp_required >= 0)
);

-- Seed level thresholds
INSERT INTO xp_level_config (level, xp_required) VALUES
    (1,      0),
    (2,    100),
    (3,    250),
    (4,    500),
    (5,   1000),
    (6,   2000),
    (7,   3500),
    (8,   5500),
    (9,   8000),
    (10, 12000),
    (15, 30000),
    (20, 70000),
    (25, 130000),
    (30, 200000)
ON CONFLICT (level) DO NOTHING;

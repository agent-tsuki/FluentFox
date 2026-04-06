-- db/seeds/xp_config.sql
-- Idempotent: safe to run multiple times with ON CONFLICT DO NOTHING.

INSERT INTO xp_reward_config (source, amount) VALUES
    ('srs_correct', 2),
    ('quiz_correct', 3),
    ('chapter_complete', 50),
    ('daily_login', 5),
    ('streak_7_days', 25),
    ('streak_30_days', 100)
ON CONFLICT (source) DO NOTHING;

INSERT INTO xp_level_config (level, xp_required) VALUES
    (1, 0),
    (2, 100),
    (3, 250),
    (4, 500),
    (5, 1000),
    (6, 2000),
    (7, 3500),
    (8, 5500),
    (9, 8000),
    (10, 12000),
    (15, 30000),
    (20, 70000),
    (25, 130000),
    (30, 200000)
ON CONFLICT (level) DO NOTHING;

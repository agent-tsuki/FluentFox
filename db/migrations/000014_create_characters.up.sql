-- 000014_create_characters.up.sql
-- Complete hiragana and katakana character sets.
-- Seeded once from db/seeds/characters.sql — never updated.
-- ~142 rows total (46 base + 25 dakuten/combination for each script).

CREATE TABLE IF NOT EXISTS characters (
    id          UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    script      character_script NOT NULL,
    -- The kana character. VARCHAR(5): most are 1 char, combined forms may be 2.
    character   VARCHAR(5)       NOT NULL,
    -- Romanisation. Example: "a", "ka", "ga"
    romaji      VARCHAR(10)      NOT NULL,
    -- Phonetic group. Examples: 'vowels', 'k-row', 'dakuten', 'combination'
    group_name  VARCHAR(50)      NULL,
    -- Canonical learning order within the script
    order_index INTEGER          NOT NULL,
    -- Object storage key for pronunciation audio
    audio_key   VARCHAR(500)     NULL,
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_characters_script_character UNIQUE (script, character)
);

CREATE INDEX idx_characters_script       ON characters (script);
CREATE INDEX idx_characters_script_order ON characters (script, order_index);
-- Used to group characters in the study UI
CREATE INDEX idx_characters_group        ON characters (script, group_name);

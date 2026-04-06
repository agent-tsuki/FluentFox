-- 000001_create_enum_types.up.sql
-- All PostgreSQL enum types must be created before any table that references them.

-- JLPT certification levels (N5=beginner, N1=advanced)
CREATE TYPE jlpt_level AS ENUM ('N5', 'N4', 'N3', 'N2', 'N1');

-- Generic learning progress state
CREATE TYPE progress_status AS ENUM ('not_started', 'in_progress', 'completed');

-- Quiz category — what kind of content was tested
CREATE TYPE quiz_type AS ENUM ('hiragana', 'katakana', 'kanji', 'grammar', 'vocabulary');

-- SRS card content type — determines which table content_id points to
CREATE TYPE srs_card_type AS ENUM ('vocabulary', 'kanji', 'character', 'concept');

-- SRS card face — direction of recall tested by this card
CREATE TYPE srs_card_face AS ENUM (
    -- vocabulary faces
    'reading_to_meaning',
    'kanji_to_reading',
    'meaning_to_kanji',
    -- kanji faces
    'kanji_to_meaning',
    'kanji_to_onyomi',
    'kanji_to_kunyomi',
    -- character (kana) faces
    'character_to_romaji',
    'romaji_to_character',
    -- concept (grammar) faces
    'pattern_to_usage',
    'usage_to_pattern'
);

-- FSRS card state — mirrors go-fsrs/v3 State type exactly
CREATE TYPE srs_card_state AS ENUM ('New', 'Learning', 'Review', 'Relearning');

-- Phonetic script
CREATE TYPE character_script AS ENUM ('hiragana', 'katakana');

-- Example sentence segment type
CREATE TYPE segment_type AS ENUM ('plain_text', 'interactive_word');

-- Shop item category — controls how frontend renders and applies the item
CREATE TYPE shop_item_type AS ENUM (
    'ui_theme',
    'cursor_effect',
    'background',
    'mascot_character',
    'streak_freeze',
    'premium_content'
);

-- XP earn event category — every XP credit has a source
CREATE TYPE xp_source_type AS ENUM (
    'chapter_completed',
    'vocab_mastered',
    'quiz_perfect_score',
    'quiz_completed',
    'streak_milestone',
    'kanji_level_cleared',
    'character_cleared',
    'srs_card_graduated',
    'daily_login',
    'admin_granted'
);

-- XP transaction direction — credit vs debit
CREATE TYPE xp_transaction_type AS ENUM ('earned', 'spent', 'refunded', 'admin_adjustment');

-- Admin permission level
CREATE TYPE admin_permission_level AS ENUM ('read', 'write', 'full');

-- Admin resource — each value maps to a DB table or logical resource
CREATE TYPE admin_resource AS ENUM (
    'chapters',
    'concepts',
    'vocabulary',
    'characters',
    'kanji_entries',
    'users',
    'shop_items',
    'xp_reward_config',
    'quiz_sessions',
    'admin_roles'
);

-- +goose Up
CREATE TYPE srs_item_type AS ENUM ('kana', 'kanji');
-- writing is reserved for later; v1 creates no writing cards.
CREATE TYPE card_facet AS ENUM ('recognition', 'meaning', 'reading', 'writing');
CREATE TYPE learning_track AS ENUM ('hiragana', 'katakana', 'kanji_n5');
CREATE TYPE session_kind AS ENUM ('lesson', 'review', 'practice', 'unit_quiz', 'skip_test', 'final_test');
CREATE TYPE question_type AS ENUM (
    'kana_mcq_romaji', 'kana_mcq_kana', 'kana_typed', 'kana_listen', 'kana_build',
    'kana_lookalike', 'kana_speed', 'kana_match', 'kana_trace',
    'kanji_meaning_mcq', 'kanji_meaning_typed', 'kanji_reading_mcq',
    'kanji_reading_typed', 'kanji_listen', 'kanji_lookalike'
);
CREATE TYPE grade_reason AS ENUM (
    'fast', 'normal', 'slow', 'mcq_cap', 'typo', 'hint', 'replay', 'wrong',
    'revealed', 'too_fast', 'idle', 'stroke_errors', 'analytics_only'
);
CREATE TYPE guide_level AS ENUM ('full', 'faded', 'none');
CREATE TYPE answer_mode AS ENUM ('choice', 'typed', 'mixed');
CREATE TYPE unit_type AS ENUM ('kana', 'kanji');
CREATE TYPE test_kind AS ENUM ('unit_quiz', 'skip', 'final');

-- +goose Down
DROP TYPE IF EXISTS test_kind;
DROP TYPE IF EXISTS unit_type;
DROP TYPE IF EXISTS answer_mode;
DROP TYPE IF EXISTS guide_level;
DROP TYPE IF EXISTS grade_reason;
DROP TYPE IF EXISTS question_type;
DROP TYPE IF EXISTS session_kind;
DROP TYPE IF EXISTS learning_track;
DROP TYPE IF EXISTS card_facet;
DROP TYPE IF EXISTS srs_item_type;

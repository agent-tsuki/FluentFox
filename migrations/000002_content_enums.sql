-- +goose Up
CREATE TYPE kana_script AS ENUM ('hiragana', 'katakana');
CREATE TYPE kana_group AS ENUM ('basic', 'dakuten', 'combo', 'foreign');
CREATE TYPE jlpt_level AS ENUM ('N5', 'N4', 'N3', 'N2', 'N1');
CREATE TYPE reading_type AS ENUM ('on', 'kun');
CREATE TYPE kun_affix AS ENUM ('prefix', 'suffix');

-- +goose Down
DROP TYPE IF EXISTS kun_affix;
DROP TYPE IF EXISTS reading_type;
DROP TYPE IF EXISTS jlpt_level;
DROP TYPE IF EXISTS kana_group;
DROP TYPE IF EXISTS kana_script;

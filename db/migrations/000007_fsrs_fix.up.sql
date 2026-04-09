ALTER TABLE fsrs_card DROP COLUMN content_type;

CREATE TYPE srs_content_type AS ENUM ('hiragana', 'katakana', 'vocabulary', 'kanji');

ALTER TABLE fsrs_card RENAME COLUMN questions_id TO content_id;
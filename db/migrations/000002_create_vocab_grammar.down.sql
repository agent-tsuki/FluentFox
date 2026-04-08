-- Drop junction tables and tables with foreign keys first
DROP TABLE IF EXISTS kanji_vocabulary;
DROP TABLE IF EXISTS chapter_chunk;
DROP TABLE IF EXISTS vocabulary;

-- Drop parent tables
DROP TABLE IF EXISTS grammar;
DROP TABLE IF EXISTS kanas;
DROP TABLE IF EXISTS kanji;

-- Drop custom types created in this migration
DROP TYPE IF EXISTS kana_type;

-- Clean up any specific indexes
DROP INDEX IF EXISTS idx_chapter_chunk_chapter_id;
DROP INDEX IF EXISTS idx_vocabulary_grammar_id;
DROP INDEX IF EXISTS idx_vocabulary_kana_id;
DROP INDEX IF EXISTS idx_vocabulary_kanji_id;
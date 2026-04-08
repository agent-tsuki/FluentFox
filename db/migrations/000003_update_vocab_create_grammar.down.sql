-- 1. Drop the table created in the UP migration
DROP TABLE IF EXISTS grammar;

-- 2. Restore the kanji_id column to the vocabulary table
-- We re-add it with the same type and reference constraint as before
ALTER TABLE vocabulary 
ADD COLUMN kanji_id UUID NULL REFERENCES kanji(id);

-- 3. (Optional) Re-create the index if it was deleted during the drop
CREATE INDEX IF NOT EXISTS idx_vocabulary_kanji_id ON vocabulary(kanji_id);
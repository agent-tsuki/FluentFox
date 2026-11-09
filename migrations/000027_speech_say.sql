-- +goose Up
-- What to say when the voice engine misreads the text as written (e.g. 国 in
-- 来月国へ read こく, or はは read as two particles): the same words with the
-- misread part in kana. Shown text is unchanged; clips are made from say
-- when it is set. Null for the rest.
ALTER TABLE chapter_passage_line ADD COLUMN say text;
ALTER TABLE chapter_word ADD COLUMN say text;

-- +goose Down
ALTER TABLE chapter_word DROP COLUMN IF EXISTS say;
ALTER TABLE chapter_passage_line DROP COLUMN IF EXISTS say;

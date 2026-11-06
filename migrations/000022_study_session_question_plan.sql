-- +goose Up
-- The questions a test asked (card and question type, in order), so the API
-- refuses answers to questions it never asked and scores the test out of
-- what it asked. NULL for lessons, reviews and practice, and for tests
-- started before this column existed.
ALTER TABLE study_session ADD COLUMN question_plan jsonb;

-- +goose Down
ALTER TABLE study_session DROP COLUMN question_plan;

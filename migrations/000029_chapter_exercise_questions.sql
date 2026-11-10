-- +goose Up
-- Chapter exercises as questions in a chapter's test. They are marked and
-- logged like any answer, but their cards never leave the new state, so they
-- stay out of lessons and reviews.
--
-- Enum values added here are only used by later transactions (PostgreSQL 12+
-- allows ADD VALUE inside a transaction on that condition).
ALTER TYPE srs_item_type ADD VALUE 'grammar_exercise';
ALTER TYPE question_type ADD VALUE 'exercise_true_false';
ALTER TYPE question_type ADD VALUE 'exercise_transform';
ALTER TYPE question_type ADD VALUE 'exercise_word_choice';
ALTER TYPE question_type ADD VALUE 'exercise_picture';
ALTER TYPE question_type ADD VALUE 'exercise_order';
ALTER TYPE question_type ADD VALUE 'exercise_listen';
ALTER TYPE question_type ADD VALUE 'exercise_reply';
ALTER TYPE question_type ADD VALUE 'exercise_reading';
ALTER TYPE chapter_exercise_kind ADD VALUE 'order';
ALTER TYPE chapter_exercise_kind ADD VALUE 'listen';
ALTER TYPE chapter_exercise_kind ADD VALUE 'reply';
ALTER TYPE chapter_exercise_kind ADD VALUE 'reading';

-- +goose Down
-- PostgreSQL cannot drop enum values; they are harmless if left behind.

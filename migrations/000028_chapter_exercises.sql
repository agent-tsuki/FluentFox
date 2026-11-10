-- +goose Up
-- A chapter's hand-written practice items for the Exercise tab, one row each:
-- true/false on a short exchange, change-the-form drills, and picture
-- questions, and sentences to drop a vocabulary word into. Each kind has its own shape, checked when the content syncs.
CREATE TYPE chapter_exercise_kind AS ENUM ('true_false', 'transform', 'picture', 'word_sentence');

CREATE TABLE chapter_exercise (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id uuid NOT NULL REFERENCES grammar_chapter (id) ON DELETE CASCADE,
    position   int NOT NULL,
    kind       chapter_exercise_kind NOT NULL,
    data       jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_chapter_exercise_chapter_position UNIQUE (chapter_id, position)
);

-- +goose Down
DROP TABLE IF EXISTS chapter_exercise;
DROP TYPE IF EXISTS chapter_exercise_kind;

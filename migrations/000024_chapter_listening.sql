-- +goose Up
-- A grammar chapter's Listening tab: short dialogues (two speakers, two
-- voices) and short passages read by one voice. Text is written with its
-- readings, e.g. "私[わたし] は 学生[がくせい] です。": a space separates words,
-- and each kanji run carries its reading in brackets, so clients can show it
-- as written, in hiragana only, or in romaji.
CREATE TYPE listening_passage_kind AS ENUM ('dialogue', 'monologue');

CREATE TABLE chapter_passage (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id uuid NOT NULL REFERENCES grammar_chapter (id) ON DELETE CASCADE,
    position   int NOT NULL,
    kind       listening_passage_kind NOT NULL,
    title      text NOT NULL,
    -- A dialogue's two speakers (written with readings); empty for a monologue.
    speakers   text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_chapter_passage_chapter_position UNIQUE (chapter_id, position),
    CONSTRAINT chk_chapter_passage_speakers CHECK (
        (kind = 'dialogue' AND cardinality(speakers) = 2) OR (kind = 'monologue' AND cardinality(speakers) = 0)
    )
);

CREATE TABLE chapter_passage_line (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    passage_id  uuid NOT NULL REFERENCES chapter_passage (id) ON DELETE CASCADE,
    position    int NOT NULL,
    -- 0 or 1: which of the dialogue's speakers says it; null in a monologue.
    speaker     smallint,
    text        text NOT NULL,
    translation text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_chapter_passage_line_passage_position UNIQUE (passage_id, position),
    CONSTRAINT chk_chapter_passage_line_speaker CHECK (speaker IS NULL OR speaker IN (0, 1))
);

-- +goose Down
DROP TABLE IF EXISTS chapter_passage_line;
DROP TABLE IF EXISTS chapter_passage;
DROP TYPE IF EXISTS listening_passage_kind;

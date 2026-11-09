-- +goose Up
-- Listening passages get a cast: each speaker has a voice (a key such as
-- "f1" or "m2" that the API maps to an engine voice), so a character sounds
-- the same in every chapter, and a dialogue may have up to four people. A
-- monologue may name its reader (one speaker, with a voice).
ALTER TABLE chapter_passage ADD COLUMN speaker_voices text[] NOT NULL DEFAULT '{}';

ALTER TABLE chapter_passage DROP CONSTRAINT chk_chapter_passage_speakers;
ALTER TABLE chapter_passage ADD CONSTRAINT chk_chapter_passage_speakers CHECK (
    (kind = 'dialogue' AND cardinality(speakers) BETWEEN 2 AND 4) OR (kind = 'monologue' AND cardinality(speakers) <= 1)
);
ALTER TABLE chapter_passage ADD CONSTRAINT chk_chapter_passage_speaker_voices CHECK (
    cardinality(speaker_voices) = 0 OR cardinality(speaker_voices) = cardinality(speakers)
);

ALTER TABLE chapter_passage_line DROP CONSTRAINT chk_chapter_passage_line_speaker;
ALTER TABLE chapter_passage_line ADD CONSTRAINT chk_chapter_passage_line_speaker CHECK (speaker IS NULL OR speaker BETWEEN 0 AND 3);

-- +goose Down
DELETE FROM chapter_passage WHERE cardinality(speakers) > 2 OR (kind = 'monologue' AND cardinality(speakers) > 0);
DELETE FROM chapter_passage_line WHERE speaker > 1;
ALTER TABLE chapter_passage_line DROP CONSTRAINT chk_chapter_passage_line_speaker;
ALTER TABLE chapter_passage_line ADD CONSTRAINT chk_chapter_passage_line_speaker CHECK (speaker IS NULL OR speaker IN (0, 1));
ALTER TABLE chapter_passage DROP CONSTRAINT chk_chapter_passage_speaker_voices;
ALTER TABLE chapter_passage DROP CONSTRAINT chk_chapter_passage_speakers;
ALTER TABLE chapter_passage ADD CONSTRAINT chk_chapter_passage_speakers CHECK (
    (kind = 'dialogue' AND cardinality(speakers) = 2) OR (kind = 'monologue' AND cardinality(speakers) = 0)
);
ALTER TABLE chapter_passage DROP COLUMN speaker_voices;

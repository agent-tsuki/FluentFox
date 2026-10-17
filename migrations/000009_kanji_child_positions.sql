-- +goose Up
-- Content sync matches kun readings and vocab on (kanji_id, position); the
-- unique index stops a re-run from ever creating a second row for a slot.
CREATE UNIQUE INDEX uq_kanji_kun_reading_kanji_position ON kanji_kun_reading (kanji_id, position);
CREATE UNIQUE INDEX uq_kanji_vocab_kanji_position ON kanji_vocab (kanji_id, position);

-- +goose Down
DROP INDEX IF EXISTS uq_kanji_vocab_kanji_position;
DROP INDEX IF EXISTS uq_kanji_kun_reading_kanji_position;

-- +goose Up
CREATE TABLE kanji_group (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    level      jlpt_level NOT NULL,
    slug       text NOT NULL,
    title      text NOT NULL,
    summary    text NOT NULL,
    position   int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_kanji_group_level_slug UNIQUE (level, slug)
);

CREATE TABLE kanji_unit (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id   uuid NOT NULL REFERENCES kanji_group (id),
    level      jlpt_level NOT NULL,
    slug       text NOT NULL,
    title      text NOT NULL,
    summary    text NOT NULL,
    position   int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_kanji_unit_level_slug UNIQUE (level, slug)
);
CREATE INDEX idx_kanji_unit_group_id ON kanji_unit (group_id);

CREATE TABLE kanji (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    char             text NOT NULL,
    level            jlpt_level NOT NULL,
    unit_id          uuid NOT NULL REFERENCES kanji_unit (id),
    position         int NOT NULL,
    meanings         text[] NOT NULL,
    synonyms         text[] NOT NULL DEFAULT '{}',
    onyomi           text[] NOT NULL DEFAULT '{}',
    key_reading_type reading_type NOT NULL,
    key_reading      text NOT NULL,
    reading_tip      text,
    mnemonic         text,
    mnemonic_draft   boolean NOT NULL DEFAULT true,
    strokes          int NOT NULL,
    grade            int,
    freq             int,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_kanji_char UNIQUE (char)
);
CREATE INDEX idx_kanji_unit_id ON kanji (unit_id);

CREATE TABLE kanji_kun_reading (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kanji_id   uuid NOT NULL REFERENCES kanji (id) ON DELETE CASCADE,
    kana       text NOT NULL,
    okurigana  text,
    affix      kun_affix,
    position   int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_kanji_kun_reading_kanji_id ON kanji_kun_reading (kanji_id);

CREATE TABLE kanji_vocab (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kanji_id   uuid NOT NULL REFERENCES kanji (id) ON DELETE CASCADE,
    word       text NOT NULL,
    reading    text NOT NULL,
    meaning    text NOT NULL,
    is_key     boolean NOT NULL DEFAULT false,
    position   int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_kanji_vocab_kanji_id ON kanji_vocab (kanji_id);

CREATE TABLE kanji_part (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    char       text NOT NULL,
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_kanji_part_char UNIQUE (char)
);

CREATE TABLE kanji_part_link (
    kanji_id   uuid NOT NULL REFERENCES kanji (id) ON DELETE CASCADE,
    part_id    uuid NOT NULL REFERENCES kanji_part (id) ON DELETE CASCADE,
    position   int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (kanji_id, part_id)
);
CREATE INDEX idx_kanji_part_link_part_id ON kanji_part_link (part_id);

CREATE TABLE kanji_look_alike (
    kanji_id       uuid NOT NULL REFERENCES kanji (id) ON DELETE CASCADE,
    other_kanji_id uuid NOT NULL REFERENCES kanji (id) ON DELETE CASCADE,
    note           text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (kanji_id, other_kanji_id),
    CONSTRAINT chk_kanji_look_alike_distinct CHECK (kanji_id <> other_kanji_id)
);
CREATE INDEX idx_kanji_look_alike_other_kanji_id ON kanji_look_alike (other_kanji_id);

-- +goose Down
DROP TABLE IF EXISTS kanji_look_alike;
DROP TABLE IF EXISTS kanji_part_link;
DROP TABLE IF EXISTS kanji_part;
DROP TABLE IF EXISTS kanji_vocab;
DROP TABLE IF EXISTS kanji_kun_reading;
DROP TABLE IF EXISTS kanji;
DROP TABLE IF EXISTS kanji_unit;
DROP TABLE IF EXISTS kanji_group;

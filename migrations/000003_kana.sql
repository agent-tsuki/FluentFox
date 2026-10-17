-- +goose Up
CREATE TABLE kana_unit (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    script       kana_script NOT NULL,
    group_key    kana_group NOT NULL,
    slug         text NOT NULL,
    title        text NOT NULL,
    summary      text NOT NULL,
    position     int NOT NULL,
    youtube_id   text,
    video_credit text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_kana_unit_script_slug UNIQUE (script, slug)
);

CREATE TABLE kana (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    script       kana_script NOT NULL,
    slug         text NOT NULL,
    character    text NOT NULL,
    romaji       text NOT NULL,
    alt_romaji   text[] NOT NULL DEFAULT '{}',
    group_key    kana_group NOT NULL,
    unit_id      uuid NOT NULL REFERENCES kana_unit (id),
    position     int NOT NULL,
    stroke_count int NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_kana_script_slug UNIQUE (script, slug),
    CONSTRAINT uq_kana_script_character UNIQUE (script, character)
);
CREATE INDEX idx_kana_unit_id ON kana (unit_id);

CREATE TABLE kana_look_alike (
    kana_id       uuid NOT NULL REFERENCES kana (id) ON DELETE CASCADE,
    other_kana_id uuid NOT NULL REFERENCES kana (id) ON DELETE CASCADE,
    note          text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (kana_id, other_kana_id),
    CONSTRAINT chk_kana_look_alike_distinct CHECK (kana_id <> other_kana_id)
);
CREATE INDEX idx_kana_look_alike_other_kana_id ON kana_look_alike (other_kana_id);

-- +goose Down
DROP TABLE IF EXISTS kana_look_alike;
DROP TABLE IF EXISTS kana;
DROP TABLE IF EXISTS kana_unit;

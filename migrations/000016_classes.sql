-- +goose Up
-- Display fields come in pairs: *_youtube is written only by the YouTube sync,
-- *_custom only by editors. The public value is custom, else youtube, so a
-- sync can never overwrite an editor's choice.
CREATE TABLE class_channels (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                   text        NOT NULL,
    platform               text        NOT NULL,
    url                    text        NOT NULL,
    youtube_channel_id     text,
    handle                 text,
    name_youtube           text,
    name_custom            text,
    avatar_url_youtube     text,
    avatar_url_custom      text,
    banner_url_youtube     text,
    banner_url_custom      text,
    description_youtube    text,
    description_custom     text,
    intro_video_id_youtube text,
    intro_video_id_custom  text,
    subscriber_count       bigint,
    video_count            bigint,
    accent_color           text,
    teaching_style         text        NOT NULL DEFAULT '',
    best_for               text[]      NOT NULL DEFAULT '{}',
    not_for                text[]      NOT NULL DEFAULT '{}',
    pros                   text[]      NOT NULL DEFAULT '{}',
    cons                   text[]      NOT NULL DEFAULT '{}',
    verdict                text        NOT NULL DEFAULT '',
    is_published           boolean     NOT NULL DEFAULT false,
    last_synced_at         timestamptz,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_class_channels_slug UNIQUE (slug),
    CONSTRAINT chk_class_channels_platform CHECK (platform IN ('youtube', 'udemy', 'website')),
    CONSTRAINT chk_class_channels_slug CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$')
);

CREATE TABLE class_courses (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug               text        NOT NULL,
    channel_id         uuid        NOT NULL REFERENCES class_channels (id),
    title              text        NOT NULL,
    level              text        NOT NULL,
    platform           text        NOT NULL,
    url                text        NOT NULL,
    playlist_id        text,
    thumbnail_url      text,
    summary            text        NOT NULL DEFAULT '',
    is_free            boolean     NOT NULL DEFAULT true,
    price_note         text        NOT NULL DEFAULT '',
    what_you_learn     jsonb       NOT NULL DEFAULT '[]',
    curriculum         jsonb       NOT NULL DEFAULT '[]',
    tags               text[]      NOT NULL DEFAULT '{}',
    is_published       boolean     NOT NULL DEFAULT false,
    is_featured        boolean     NOT NULL DEFAULT false,
    sort_order         integer     NOT NULL DEFAULT 0,
    total_duration_sec integer     NOT NULL DEFAULT 0,
    last_synced_at     timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_class_courses_slug UNIQUE (slug),
    CONSTRAINT chk_class_courses_level CHECK (level IN ('N5', 'N4', 'N3', 'N2', 'N1')),
    CONSTRAINT chk_class_courses_platform CHECK (platform IN ('youtube', 'udemy', 'website')),
    CONSTRAINT chk_class_courses_slug CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$')
);

CREATE INDEX idx_class_courses_channel ON class_courses (channel_id);

-- Replaced as a whole on every playlist sync.
CREATE TABLE class_videos (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id        uuid    NOT NULL REFERENCES class_courses (id) ON DELETE CASCADE,
    youtube_video_id text    NOT NULL,
    title            text    NOT NULL,
    thumbnail_url    text    NOT NULL DEFAULT '',
    duration_sec     integer NOT NULL DEFAULT 0,
    position         integer NOT NULL,
    CONSTRAINT uq_class_videos_course_position UNIQUE (course_id, position)
);

-- No foreign keys: the log must outlive a deleted course or user.
CREATE TABLE class_change_log (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    uuid        NOT NULL,
    entity_type text        NOT NULL,
    entity_id   uuid        NOT NULL,
    field       text        NOT NULL,
    old_value   text,
    new_value   text,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_class_change_log_entity ON class_change_log (entity_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS class_change_log;
DROP TABLE IF EXISTS class_videos;
DROP TABLE IF EXISTS class_courses;
DROP TABLE IF EXISTS class_channels;

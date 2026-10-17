-- +goose Up
-- target_item_id and chosen_item_id point at kana or kanji depending on
-- item_type, so they have no foreign keys.
CREATE TABLE user_confusion (
    user_id        uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    item_type      srs_item_type NOT NULL,
    target_item_id uuid NOT NULL,
    chosen_item_id uuid NOT NULL,
    times          int NOT NULL DEFAULT 1,
    last_at        timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, item_type, target_item_id, chosen_item_id)
);

CREATE TABLE daily_activity (
    user_id          uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    local_date       date NOT NULL,
    reviews_done     int NOT NULL DEFAULT 0,
    new_learned      int NOT NULL DEFAULT 0,
    correct_count    int NOT NULL DEFAULT 0,
    again_count      int NOT NULL DEFAULT 0,
    active_ms        bigint NOT NULL DEFAULT 0,
    due_at_day_start int NOT NULL DEFAULT 0,
    goal_met         boolean NOT NULL DEFAULT false,
    PRIMARY KEY (user_id, local_date)
);

-- unit_id points at kana_unit or kanji_unit depending on unit_type.
CREATE TABLE unit_progress (
    user_id       uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    unit_type     unit_type NOT NULL,
    unit_id       uuid NOT NULL,
    watched_at    timestamptz,
    traced_at     timestamptz,
    written_at    timestamptz,
    practised_at  timestamptz,
    quiz_best_pct smallint,
    completed_at  timestamptz,
    PRIMARY KEY (user_id, unit_type, unit_id),
    CONSTRAINT chk_unit_progress_quiz_best_pct CHECK (quiz_best_pct BETWEEN 0 AND 100)
);

CREATE TABLE test_result (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    session_id uuid NOT NULL REFERENCES study_session (id) ON DELETE CASCADE,
    kind       test_kind NOT NULL,
    track      learning_track NOT NULL,
    scope      text NOT NULL,
    score      int NOT NULL,
    total      int NOT NULL,
    passed     boolean NOT NULL,
    active_ms  bigint NOT NULL DEFAULT 0,
    taken_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_test_result_score CHECK (score BETWEEN 0 AND total)
);
CREATE INDEX idx_test_result_user_taken_at ON test_result (user_id, taken_at DESC);
CREATE INDEX idx_test_result_session_id ON test_result (session_id);

CREATE TABLE kanji_note (
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kanji_id   uuid NOT NULL REFERENCES kanji (id) ON DELETE CASCADE,
    text       varchar(1000) NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, kanji_id)
);

-- +goose Down
DROP TABLE IF EXISTS kanji_note;
DROP TABLE IF EXISTS test_result;
DROP TABLE IF EXISTS unit_progress;
DROP TABLE IF EXISTS daily_activity;
DROP TABLE IF EXISTS user_confusion;

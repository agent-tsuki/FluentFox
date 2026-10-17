-- state is 0-3 (new, learning, review, relearning) and rating 1-4 (Again, Hard,
-- Good, Easy) to match go-fsrs. item_id points at kana or kanji depending on
-- item_type, so it has no foreign key.

-- +goose Up
CREATE TABLE srs_card (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    item_type      srs_item_type NOT NULL,
    item_id        uuid NOT NULL,
    facet          card_facet NOT NULL,
    state          smallint NOT NULL DEFAULT 0,
    due            timestamptz NOT NULL,
    stability      double precision NOT NULL DEFAULT 0,
    difficulty     double precision NOT NULL DEFAULT 0,
    reps           int NOT NULL DEFAULT 0,
    lapses         int NOT NULL DEFAULT 0,
    learning_step  smallint NOT NULL DEFAULT 0,
    scheduled_days int NOT NULL DEFAULT 0,
    last_review_at timestamptz,
    is_leech       boolean NOT NULL DEFAULT false,
    introduced_at  timestamptz NOT NULL DEFAULT now(),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_srs_card_user_item_facet UNIQUE (user_id, item_type, item_id, facet),
    CONSTRAINT chk_srs_card_state CHECK (state BETWEEN 0 AND 3)
);
-- New cards are picked by lesson order, not by due date, so they stay out of the
-- review queue index.
CREATE INDEX idx_srs_card_user_due ON srs_card (user_id, due) WHERE state <> 0;

CREATE TABLE study_session (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind          session_kind NOT NULL,
    track         learning_track,
    scope         text,
    started_at    timestamptz NOT NULL DEFAULT now(),
    completed_at  timestamptz,
    answer_count  int NOT NULL DEFAULT 0,
    correct_count int NOT NULL DEFAULT 0,
    active_ms     bigint NOT NULL DEFAULT 0
);
CREATE INDEX idx_study_session_user_started_at ON study_session (user_id, started_at DESC);

-- Append only.
CREATE TABLE review_log (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    card_id               uuid NOT NULL REFERENCES srs_card (id) ON DELETE CASCADE,
    session_id            uuid NOT NULL REFERENCES study_session (id) ON DELETE CASCADE,
    client_event_id       uuid NOT NULL,
    question_type         question_type NOT NULL,
    answered_at           timestamptz NOT NULL,
    -- Sent by the client.
    shown_at              timestamptz NOT NULL,
    duration_ms           int NOT NULL,
    idle_ms               int NOT NULL DEFAULT 0,
    first_input_ms        int,
    attempts              smallint NOT NULL DEFAULT 1,
    answer_text           varchar(64),
    chosen_option         text,
    options               text[],
    hints_used            smallint NOT NULL DEFAULT 0,
    answer_revealed       boolean NOT NULL DEFAULT false,
    audio_replays         smallint NOT NULL DEFAULT 0,
    audio_ms              int,
    stroke_errors         smallint,
    stroke_count          smallint,
    guide_level           guide_level,
    -- Computed by the server.
    is_correct            boolean NOT NULL,
    typo_distance         smallint,
    active_ms             int NOT NULL,
    rating                smallint,
    grade_reason          grade_reason NOT NULL,
    counts_for_srs        boolean NOT NULL,
    state_before          smallint NOT NULL,
    retrievability_before double precision,
    stability_before      double precision,
    stability_after       double precision,
    difficulty_before     double precision,
    difficulty_after      double precision,
    elapsed_days          int NOT NULL DEFAULT 0,
    scheduled_days        int NOT NULL DEFAULT 0,
    due_after             timestamptz,
    -- Batches are retried by the client, so the event id makes a resend a no-op.
    CONSTRAINT uq_review_log_user_client_event UNIQUE (user_id, client_event_id),
    CONSTRAINT chk_review_log_rating CHECK (rating BETWEEN 1 AND 4),
    CONSTRAINT chk_review_log_state_before CHECK (state_before BETWEEN 0 AND 3)
);
CREATE INDEX idx_review_log_user_question_answered ON review_log (user_id, question_type, answered_at DESC);
CREATE INDEX idx_review_log_card_answered ON review_log (card_id, answered_at);
CREATE INDEX idx_review_log_session_id ON review_log (session_id);

-- +goose Down
DROP TABLE IF EXISTS review_log;
DROP TABLE IF EXISTS study_session;
DROP TABLE IF EXISTS srs_card;

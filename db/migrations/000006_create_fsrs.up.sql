-- db/migrations/000006_create_fsrs.up.sql


CREATE TABLE fsrs_card (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- FSRS Data 
    due            TIMESTAMPTZ NOT NULL,
    stability      DOUBLE PRECISION NOT NULL,
    difficulty     DOUBLE PRECISION NOT NULL,
    elapsed_days   INTEGER NOT NULL,
    scheduled_days INTEGER NOT NULL,
    reps           INTEGER NOT NULL DEFAULT 0,
    lapses         INTEGER NOT NULL DEFAULT 0,
    state          VARCHAR NOT NULL,
    last_review    TIMESTAMPTZ,

    -- Polymorphic Reference
    content_type   quiz_type NOT NULL,
    questions_id     UUID NOT NULL,

    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE review_log (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id        UUID NOT NULL REFERENCES fsrs_card(id) ON DELETE CASCADE,

    rating INTEGER NOT NULL CHECK (rating > 0 AND rating < 5),
    scheduled_days INTEGER NOT NULL,
    elapsed_days   INTEGER NOT NULL,
    review_at      TIMESTAMPTZ DEFAULT NOW(),
    state          VARCHAR NOT NULL,

    created_at     TIMESTAMPTZ DEFAULT NOW()
);

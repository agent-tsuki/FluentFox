-- +goose Up
-- Which in-app tutorials each learner has seen. No row means pending: a new
-- account gets the tour without a write at sign-up. version is the tutorial
-- version the status is for, so the frontend can show a redesigned tour again.
CREATE TYPE tutorial_status AS ENUM ('pending', 'skipped', 'completed');

CREATE TABLE user_tutorial (
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tutorial_key text NOT NULL,
    status       tutorial_status NOT NULL DEFAULT 'pending',
    version      int NOT NULL DEFAULT 1,
    last_step    int,
    completed_at timestamptz,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, tutorial_key),
    CONSTRAINT chk_user_tutorial_key_len CHECK (char_length(tutorial_key) BETWEEN 1 AND 40),
    CONSTRAINT chk_user_tutorial_version CHECK (version >= 1),
    CONSTRAINT chk_user_tutorial_last_step CHECK (last_step IS NULL OR last_step >= 0)
);

-- Learners who joined before the welcome tour existed already know the app:
-- record it as skipped so it never interrupts them.
INSERT INTO user_tutorial (user_id, tutorial_key, status, version)
SELECT id, 'welcome', 'skipped', 1 FROM users
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS user_tutorial;
DROP TYPE IF EXISTS tutorial_status;

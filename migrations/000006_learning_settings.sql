-- The row is created lazily with these defaults on first read. The user's time
-- zone is not stored here; it comes from users_settings.current_time_zone.

-- +goose Up
CREATE TABLE learning_settings (
    user_id             uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    desired_retention   numeric(3,2) NOT NULL DEFAULT 0.90,
    new_kana_per_day    int NOT NULL DEFAULT 15,
    new_kanji_per_day   int NOT NULL DEFAULT 10,
    max_reviews_per_day int NOT NULL DEFAULT 200,
    session_size        int NOT NULL DEFAULT 20,
    lesson_size         int NOT NULL DEFAULT 5,
    daily_goal_answers  int NOT NULL DEFAULT 20,
    day_starts_at_hour  smallint NOT NULL DEFAULT 4,
    answer_mode         answer_mode NOT NULL DEFAULT 'mixed',
    show_retention      boolean NOT NULL DEFAULT true,
    -- Per-user FSRS optimizer output; NULL means the library defaults.
    fsrs_weights        double precision[],
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_learning_settings_desired_retention CHECK (desired_retention BETWEEN 0.70 AND 0.97),
    CONSTRAINT chk_learning_settings_new_kana_per_day CHECK (new_kana_per_day BETWEEN 0 AND 100),
    CONSTRAINT chk_learning_settings_new_kanji_per_day CHECK (new_kanji_per_day BETWEEN 0 AND 50),
    CONSTRAINT chk_learning_settings_max_reviews_per_day CHECK (max_reviews_per_day BETWEEN 10 AND 1000),
    CONSTRAINT chk_learning_settings_session_size CHECK (session_size BETWEEN 5 AND 100),
    CONSTRAINT chk_learning_settings_lesson_size CHECK (lesson_size BETWEEN 1 AND 20),
    CONSTRAINT chk_learning_settings_daily_goal_answers CHECK (daily_goal_answers BETWEEN 5 AND 200),
    CONSTRAINT chk_learning_settings_day_starts_at_hour CHECK (day_starts_at_hour BETWEEN 0 AND 23)
);

-- +goose Down
DROP TABLE IF EXISTS learning_settings;

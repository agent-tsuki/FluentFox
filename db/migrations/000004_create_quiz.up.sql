-- db/migrations/000004_create_quiz.up.sql

-- Create quiz type enum
CREATE TYPE quiz_type AS ENUM ('hiragana', 'katakana', 'vocabulary', 'grammar');

-- Tables
CREATE TABLE quiz_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    target_level  jlpt_level NOT NULL,
    

    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE quiz_questions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    content_type           quiz_type NOT NULL,
    content_id              VARCHAR NOT NULL,

    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE quiz_answers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id    UUID NULL REFERENCES quiz_questions(id),
    session_id       UUID NULL REFERENCES quiz_sessions(id),

    selected    VARCHAR NOT NULL,
    correct_ans VARCHAR NOT NULL,
    

    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

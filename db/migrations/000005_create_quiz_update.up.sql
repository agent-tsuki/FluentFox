-- Fix quiz_questions
ALTER TABLE quiz_questions
    ALTER COLUMN content_id TYPE UUID USING content_id::UUID,
    ADD COLUMN question   TEXT NOT NULL DEFAULT '',
    ADD COLUMN options    JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN correct_id VARCHAR NOT NULL DEFAULT '';

-- Fix quiz_answers  
ALTER TABLE quiz_answers
    ALTER COLUMN question_id SET NOT NULL,
    ALTER COLUMN session_id  SET NOT NULL;

-- Fix quiz_sessions
ALTER TABLE quiz_sessions
    ADD COLUMN total_questions INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN correct_count   INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN completed_at    TIMESTAMPTZ NULL;
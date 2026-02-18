-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Custom ENUM for difficulty perception
DO $$ BEGIN
    CREATE TYPE difficulty_level AS ENUM ('easy', 'ok', 'hard');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Feedback table
CREATE TABLE IF NOT EXISTS feedback (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id           UUID NOT NULL,
    message              TEXT,
    perceived_difficulty difficulty_level NOT NULL,
    strategy_snapshot    JSONB NOT NULL,
    created_at           TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_feedback_student_id ON feedback(student_id);
CREATE INDEX IF NOT EXISTS idx_feedback_created_at ON feedback(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feedback_strategy_snapshot ON feedback USING GIN (strategy_snapshot);

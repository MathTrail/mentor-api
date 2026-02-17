-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Custom ENUM for difficulty perception
CREATE TYPE difficulty_level AS ENUM ('easy', 'ok', 'hard');

-- Main feedback table
CREATE TABLE IF NOT EXISTS feedback (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id           UUID NOT NULL,
    message              TEXT,
    perceived_difficulty difficulty_level NOT NULL,
    strategy_snapshot    JSONB NOT NULL,
    created_at           TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_feedback_student_id ON feedback(student_id);
CREATE INDEX IF NOT EXISTS idx_feedback_created_at ON feedback(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feedback_difficulty ON feedback(perceived_difficulty);

-- GIN index for JSONB queries (analytics)
CREATE INDEX IF NOT EXISTS idx_feedback_strategy_snapshot ON feedback USING GIN (strategy_snapshot);

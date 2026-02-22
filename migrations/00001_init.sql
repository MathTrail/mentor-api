-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE difficulty_level AS ENUM ('easy', 'ok', 'hard');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS feedback (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id           UUID NOT NULL,
    message              TEXT,
    perceived_difficulty difficulty_level NOT NULL,
    strategy_snapshot    JSONB NOT NULL,
    created_at           TIMESTAMP NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_feedback_student_id ON feedback(student_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_feedback_created_at ON feedback(created_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_feedback_strategy_snapshot ON feedback USING GIN (strategy_snapshot);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feedback_strategy_snapshot;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feedback_created_at;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feedback_student_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS feedback;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TYPE IF EXISTS difficulty_level;
-- +goose StatementEnd

-- +goose StatementBegin
DROP EXTENSION IF EXISTS "uuid-ossp";
-- +goose StatementEnd

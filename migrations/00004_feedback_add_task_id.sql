-- +goose Up
-- +goose StatementBegin
ALTER TABLE feedback ADD COLUMN task_id TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_feedback_task_id ON feedback(task_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feedback_task_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE feedback DROP COLUMN task_id;
-- +goose StatementEnd

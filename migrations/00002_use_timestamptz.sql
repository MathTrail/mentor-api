-- +goose Up
-- +goose StatementBegin
ALTER TABLE feedback
    ALTER COLUMN created_at TYPE TIMESTAMPTZ
    USING created_at AT TIME ZONE 'UTC';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE feedback
    ALTER COLUMN created_at TYPE TIMESTAMP
    USING created_at AT TIME ZONE 'UTC';
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
ALTER TABLE recommendations
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE recommendations DROP COLUMN updated_at;
-- +goose StatementEnd

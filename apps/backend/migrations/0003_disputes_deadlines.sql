-- +goose Up
-- +goose StatementBegin
ALTER TABLE disputes
    ADD COLUMN IF NOT EXISTS ends_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN IF NOT EXISTS next_deadline TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE disputes
SET
    ends_at = created_at + INTERVAL '1 day',
    next_deadline = created_at + INTERVAL '1 day';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE disputes
    DROP COLUMN IF EXISTS next_deadline,
    DROP COLUMN IF EXISTS ends_at;
-- +goose StatementEnd

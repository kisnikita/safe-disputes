-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ALTER COLUMN created_at TYPE TIMESTAMPTZ;

ALTER TABLE disputes
    ALTER COLUMN created_at TYPE TIMESTAMPTZ,
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ,
    ALTER COLUMN ends_at TYPE TIMESTAMPTZ,
    ALTER COLUMN next_deadline TYPE TIMESTAMPTZ;

ALTER TABLE investigations
    ALTER COLUMN created_at TYPE TIMESTAMPTZ,
    ALTER COLUMN ends_at TYPE TIMESTAMPTZ;

ALTER TABLE evidence
    ALTER COLUMN created_at TYPE TIMESTAMPTZ,
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE evidence
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN created_at TYPE TIMESTAMP;

ALTER TABLE investigations
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN ends_at TYPE TIMESTAMP;

ALTER TABLE disputes
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN next_deadline TYPE TIMESTAMP,
    ALTER COLUMN ends_at TYPE TIMESTAMP;

ALTER TABLE users
    ALTER COLUMN created_at TYPE TIMESTAMP;
-- +goose StatementEnd

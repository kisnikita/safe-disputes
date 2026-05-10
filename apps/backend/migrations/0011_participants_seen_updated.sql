-- +goose Up
-- +goose StatementBegin
ALTER TABLE participants
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS seen_at TIMESTAMPTZ NULL;

UPDATE participants
SET seen_at = updated_at;

CREATE INDEX IF NOT EXISTS idx_participants_user_updated_id
    ON participants (user_id, updated_at);

ALTER TABLE jurors
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS seen_at TIMESTAMPTZ NULL;

UPDATE jurors
SET seen_at = updated_at;

CREATE INDEX IF NOT EXISTS idx_jurors_user_updated_id
    ON jurors (user_id, updated_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_participants_user_updated_id;
DROP INDEX IF EXISTS idx_jurors_user_updated_id;

ALTER TABLE participants
    DROP COLUMN IF EXISTS seen_at,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE jurors
    DROP COLUMN IF EXISTS seen_at,
    DROP COLUMN IF EXISTS updated_at;
-- +goose StatementEnd

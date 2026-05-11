-- +goose Up
-- +goose StatementBegin
ALTER TABLE participants
    ADD COLUMN IF NOT EXISTS is_creator BOOLEAN NOT NULL DEFAULT FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS participants_unique_dispute_creator_true
    ON participants (dispute_id)
    WHERE is_creator = TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS participants_unique_dispute_creator_true;
ALTER TABLE participants DROP COLUMN IF EXISTS is_creator;
-- +goose StatementEnd

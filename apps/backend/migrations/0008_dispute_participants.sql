-- +goose Up
-- +goose StatementBegin
ALTER TABLE user2dispute RENAME TO dispute_participants;
CREATE UNIQUE INDEX IF NOT EXISTS dispute_participants_unique_dispute_user
    ON dispute_participants (dispute_id, user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS dispute_participants_unique_dispute_user;

ALTER TABLE IF EXISTS dispute_participants RENAME TO user2dispute;
-- +goose StatementEnd

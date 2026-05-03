-- +goose Up
-- +goose StatementBegin
ALTER TABLE user2dispute RENAME TO participants;
ALTER INDEX IF EXISTS user2dispute_pkey RENAME TO participants_pkey;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'user2dispute_user_id_fkey'
    ) THEN
        ALTER TABLE participants
            RENAME CONSTRAINT user2dispute_user_id_fkey TO participants_user_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'user2dispute_dispute_id_fkey'
    ) THEN
        ALTER TABLE participants
            RENAME CONSTRAINT user2dispute_dispute_id_fkey TO participants_dispute_id_fkey;
    END IF;
END
$$;

CREATE UNIQUE INDEX IF NOT EXISTS participants_unique_dispute_user
    ON participants (dispute_id, user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS participants_unique_dispute_user;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'participants_user_id_fkey'
    ) THEN
        ALTER TABLE participants
            RENAME CONSTRAINT participants_user_id_fkey TO user2dispute_user_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'participants_dispute_id_fkey'
    ) THEN
        ALTER TABLE participants
            RENAME CONSTRAINT participants_dispute_id_fkey TO user2dispute_dispute_id_fkey;
    END IF;
END
$$;

ALTER INDEX IF EXISTS participants_pkey RENAME TO user2dispute_pkey;
ALTER TABLE IF EXISTS participants RENAME TO user2dispute;
-- +goose StatementEnd

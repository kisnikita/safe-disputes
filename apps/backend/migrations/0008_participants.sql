-- +goose Up
-- +goose StatementBegin
ALTER TABLE user2dispute RENAME TO participants;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'user2dispute_pkey'
    ) THEN
        ALTER TABLE participants
            RENAME CONSTRAINT user2dispute_pkey TO participants_pkey;
    END IF;

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
        WHERE conname = 'participants_pkey'
    ) THEN
        ALTER TABLE participants
            RENAME CONSTRAINT participants_pkey TO user2dispute_pkey;
    END IF;

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

ALTER TABLE IF EXISTS participants RENAME TO user2dispute;
-- +goose StatementEnd

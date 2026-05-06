-- +goose Up
-- +goose StatementBegin
ALTER TABLE IF EXISTS user2investigation RENAME TO jurors;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'user2investigation_pkey'
    ) THEN
        ALTER TABLE jurors
            RENAME CONSTRAINT user2investigation_pkey TO jurors_pkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'user2investigation_user_id_fkey'
    ) THEN
        ALTER TABLE jurors
            RENAME CONSTRAINT user2investigation_user_id_fkey TO jurors_user_id_fkey;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'jurors_investigation_id_fkey'
    ) THEN
        ALTER TABLE jurors
            ADD CONSTRAINT jurors_investigation_id_fkey
            FOREIGN KEY (investigation_id) REFERENCES investigations(id);
    END IF;
END
$$;

CREATE UNIQUE INDEX IF NOT EXISTS jurors_unique_investigation_user
    ON jurors (investigation_id, user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS jurors_unique_investigation_user;

ALTER TABLE IF EXISTS jurors
    DROP CONSTRAINT IF EXISTS jurors_investigation_id_fkey;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'jurors_pkey'
    ) THEN
        ALTER TABLE jurors
            RENAME CONSTRAINT jurors_pkey TO user2investigation_pkey;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'jurors_user_id_fkey'
    ) THEN
        ALTER TABLE jurors
            RENAME CONSTRAINT jurors_user_id_fkey TO user2investigation_user_id_fkey;
    END IF;
END
$$;

ALTER TABLE IF EXISTS jurors RENAME TO user2investigation;
-- +goose StatementEnd

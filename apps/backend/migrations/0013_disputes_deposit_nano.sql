-- +goose Up
-- +goose StatementBegin
ALTER TABLE disputes
    ADD COLUMN IF NOT EXISTS deposit_nano BIGINT;

UPDATE disputes
SET deposit_nano = GREATEST(amount_nano / 10, 200000000)
WHERE deposit_nano IS NULL;

ALTER TABLE disputes
    ALTER COLUMN deposit_nano SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'disputes_deposit_nano_positive'
    ) THEN
        ALTER TABLE disputes
            ADD CONSTRAINT disputes_deposit_nano_positive CHECK (deposit_nano > 0);
    END IF;
END$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE disputes
    DROP CONSTRAINT IF EXISTS disputes_deposit_nano_positive,
    DROP COLUMN IF EXISTS deposit_nano;
-- +goose StatementEnd

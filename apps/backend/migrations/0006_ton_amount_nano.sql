-- +goose Up
-- +goose StatementBegin
ALTER TABLE disputes
    ADD COLUMN IF NOT EXISTS amount_nano BIGINT;

UPDATE disputes
SET amount_nano = ROUND(amount * 1000000000)::BIGINT
WHERE amount_nano IS NULL;

ALTER TABLE disputes
    ALTER COLUMN amount_nano SET NOT NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS minimum_dispute_amount_nano BIGINT;

UPDATE users
SET minimum_dispute_amount_nano = ROUND(minimum_dispute_amount * 1000000000)::BIGINT
WHERE minimum_dispute_amount_nano IS NULL;

ALTER TABLE users
    ALTER COLUMN minimum_dispute_amount_nano SET NOT NULL,
    ALTER COLUMN minimum_dispute_amount_nano SET DEFAULT 0;

ALTER TABLE disputes
    DROP COLUMN IF EXISTS amount;

ALTER TABLE users
    DROP COLUMN IF EXISTS minimum_dispute_amount;

UPDATE disputes
SET cryptocurrency = 'TON'
WHERE cryptocurrency IS NULL OR cryptocurrency <> 'TON';

ALTER TABLE disputes
    ALTER COLUMN cryptocurrency SET DEFAULT 'TON',
    ALTER COLUMN cryptocurrency SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'disputes_amount_nano_positive'
    ) THEN
        ALTER TABLE disputes
            ADD CONSTRAINT disputes_amount_nano_positive CHECK (amount_nano > 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_minimum_dispute_amount_nano_non_negative'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_minimum_dispute_amount_nano_non_negative CHECK (minimum_dispute_amount_nano >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'disputes_cryptocurrency_ton_only'
    ) THEN
        ALTER TABLE disputes
            ADD CONSTRAINT disputes_cryptocurrency_ton_only CHECK (cryptocurrency = 'TON');
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE disputes
    DROP CONSTRAINT IF EXISTS disputes_cryptocurrency_ton_only,
    DROP CONSTRAINT IF EXISTS disputes_amount_nano_positive,
    ALTER COLUMN cryptocurrency DROP DEFAULT;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_minimum_dispute_amount_nano_non_negative,
    ALTER COLUMN minimum_dispute_amount_nano DROP DEFAULT;

ALTER TABLE disputes
    ADD COLUMN IF NOT EXISTS amount NUMERIC;

UPDATE disputes
SET amount = (amount_nano::numeric / 1000000000)
WHERE amount IS NULL;

ALTER TABLE disputes
    ALTER COLUMN amount SET NOT NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS minimum_dispute_amount NUMERIC;

UPDATE users
SET minimum_dispute_amount = (minimum_dispute_amount_nano::numeric / 1000000000)
WHERE minimum_dispute_amount IS NULL;

ALTER TABLE users
    ALTER COLUMN minimum_dispute_amount SET NOT NULL,
    ALTER COLUMN minimum_dispute_amount SET DEFAULT 0;

ALTER TABLE disputes
    DROP COLUMN IF EXISTS amount_nano;

ALTER TABLE users
    DROP COLUMN IF EXISTS minimum_dispute_amount_nano;
-- +goose StatementEnd

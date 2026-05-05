-- +goose Up
-- +goose StatementBegin
ALTER TABLE evidence RENAME TO evidences;

ALTER TABLE evidences
    ADD COLUMN IF NOT EXISTS participant_id uuid;

UPDATE evidences e
SET participant_id = p.id
FROM participants p
WHERE p.dispute_id = e.dispute_id
  AND p.user_id = e.user_id
  AND e.participant_id IS NULL;

ALTER TABLE evidences
    ALTER COLUMN participant_id SET NOT NULL;

ALTER TABLE evidences
    ADD CONSTRAINT evidences_participant_id_fkey
    FOREIGN KEY (participant_id) REFERENCES participants(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX evidences_unique_participant_id
    ON evidences (participant_id);

ALTER TABLE evidences
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS user_id,
    DROP COLUMN IF EXISTS dispute_id;    
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE evidences
    ADD COLUMN IF NOT EXISTS dispute_id uuid,
    ADD COLUMN IF NOT EXISTS user_id uuid,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE evidences e
SET dispute_id = p.dispute_id,
    user_id = p.user_id
FROM participants p
WHERE p.id = e.participant_id
  AND (e.dispute_id IS NULL OR e.user_id IS NULL);

ALTER TABLE evidences
    ALTER COLUMN dispute_id SET NOT NULL,
    ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE evidences
    ADD CONSTRAINT evidences_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id),
    ADD CONSTRAINT evidences_dispute_id_fkey FOREIGN KEY (dispute_id) REFERENCES disputes(id);

DROP INDEX IF EXISTS evidences_unique_participant_id;
ALTER TABLE evidences DROP CONSTRAINT IF EXISTS evidences_participant_id_fkey;
ALTER TABLE evidences DROP COLUMN IF EXISTS participant_id;
ALTER TABLE evidences RENAME TO evidence;
-- +goose StatementEnd

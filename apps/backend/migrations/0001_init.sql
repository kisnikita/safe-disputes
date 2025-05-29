-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS USERS (
    id uuid PRIMARY KEY,
    username text NOT NULL,
    chat_id NUMERIC NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    notification_enabled BOOLEAN DEFAULT FALSE,
    dispute_readiness BOOLEAN DEFAULT TRUE,
    minimum_dispute_amount NUMERIC DEFAULT 0,
    rating INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_users_username ON USERS (username);

CREATE TABLE IF NOT EXISTS DISPUTES (
    id uuid PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    cryptocurrency VARCHAR(50),
    amount NUMERIC,
    image_data   BYTEA   NULL,
    image_type   TEXT    NULL
);

CREATE TABLE IF NOT EXISTS USER2DISPUTE (
    id uuid,
    user_id uuid,
    dispute_id uuid,
    vote BOOLEAN DEFAULT FALSE,
    result TEXT,
    status TEXT,
    claim BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (user_id) REFERENCES USERS(id),
    FOREIGN KEY (dispute_id) REFERENCES DISPUTES(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS USER2DISPUTE;
DROP TABLE IF EXISTS DISPUTES CASCADE;
DROP TABLE IF EXISTS USERS CASCADE;
-- +goose StatementEnd

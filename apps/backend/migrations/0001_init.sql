-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS USERS (
    id uuid,
    username text NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    refresh_token VARCHAR(2048),
    notification_enabled BOOLEAN DEFAULT TRUE,
    dispute_readiness BOOLEAN DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS idx_users_username ON USERS (username);
CREATE INDEX IF NOT EXISTS idx_users_refresh_token ON USERS (refresh_token);

CREATE TABLE IF NOT EXISTS DISPUTES (
    id uuid,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) DEFAULT 'pending',
    cryptocurrency VARCHAR(50),
    amount NUMERIC
);

CREATE TABLE IF NOT EXISTS USER2DISPUTES (
    id uuid,
    user_id uuid,
    dispute_id uuid,
    result VARCHAR(50),
    FOREIGN KEY (user_id) REFERENCES USERS(id),
    FOREIGN KEY (dispute_id) REFERENCES DISPUTES(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS USER2DISPUTES;
DROP TABLE IF EXISTS DISPUTES;
DROP TABLE IF EXISTS USERS;
-- +goose StatementEnd

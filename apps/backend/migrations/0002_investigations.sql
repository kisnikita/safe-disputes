-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS INVESTIGATIONS
(
    id         uuid PRIMARY KEY,
    dispute_id uuid    NOT NULL,
    total      INTEGER NOT NULL,
    p1         INTEGER NOT NULL,
    p2         INTEGER NOT NULL,
    draw       INTEGER NOT NULL,
    status     TEXT      DEFAULT 'open',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ends_at    TIMESTAMP,
    title      TEXT,

    FOREIGN KEY (dispute_id) REFERENCES DISPUTES (id)
);

CREATE TABLE IF NOT EXISTS USER2INVESTIGATION
(
    id               uuid,
    user_id          uuid,
    investigation_id uuid,
    vote             TEXT,
    result           TEXT,
    FOREIGN KEY (user_id) REFERENCES USERS (id)
);

CREATE TABLE IF NOT EXISTS EVIDENCE
(
    id          uuid PRIMARY KEY,
    dispute_id  uuid  NOT NULL,
    user_id     uuid  NOT NULL,
    description TEXT,
    image_data  BYTEA NULL,
    image_type  TEXT  NULL,

    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES USERS (id),
    FOREIGN KEY (dispute_id) REFERENCES DISPUTES (id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS USER2INVESTIGATION;
DROP TABLE IF EXISTS EVIDENCE;
DROP TABLE IF EXISTS INVESTIGATIONS CASCADE;
-- +goose StatementEnd

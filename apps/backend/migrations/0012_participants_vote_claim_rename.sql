-- +goose Up
-- +goose StatementBegin
ALTER TABLE participants RENAME COLUMN vote TO is_win;
ALTER TABLE participants RENAME COLUMN claim TO is_claimable;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE participants RENAME COLUMN is_win TO vote;
ALTER TABLE participants RENAME COLUMN is_claimable TO claim;
-- +goose StatementEnd

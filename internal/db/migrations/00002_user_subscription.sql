-- +goose Up

ALTER TABLE users ADD COLUMN sub_token TEXT;

UPDATE users
SET sub_token = lower(hex(randomblob(16)))
WHERE sub_token IS NULL;

CREATE UNIQUE INDEX users_sub_token_idx ON users(sub_token);

-- +goose Down

DROP INDEX IF EXISTS users_sub_token_idx;


-- +goose Up
ALTER TABLE users
ADD hashed_password TEXT DEFAULT 'unset' NOT NULL;

-- +goose Down
REMOVE hashed_password FROM users;
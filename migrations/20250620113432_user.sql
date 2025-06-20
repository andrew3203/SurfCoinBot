-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    username TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('athlete', 'coach'))
);

-- +goose Down
DROP TABLE IF EXISTS user;

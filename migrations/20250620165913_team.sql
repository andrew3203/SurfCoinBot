-- +goose Up
CREATE TABLE IF NOT EXISTS team (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE IF EXISTS team;

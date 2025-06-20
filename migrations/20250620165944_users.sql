-- +goose Up
ALTER TABLE users
ADD COLUMN team_id INTEGER REFERENCES team(id);

-- +goose Down
ALTER TABLE users
DROP COLUMN IF EXISTS team_id;

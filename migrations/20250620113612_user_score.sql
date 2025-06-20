-- +goose Up
CREATE TABLE IF NOT EXISTS user_score (
    user_id BIGINT PRIMARY KEY REFERENCES users(id),
    score INT NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS user_score;

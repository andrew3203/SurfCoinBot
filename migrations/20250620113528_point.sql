-- +goose Up
CREATE TABLE IF NOT EXISTS point (
    id SERIAL PRIMARY KEY,
    from_id BIGINT REFERENCES users(id),
    amount INT NOT NULL,
    reason TEXT NOT NULL,
    pending BOOLEAN DEFAULT true
);

-- +goose Down
DROP TABLE IF EXISTS point;

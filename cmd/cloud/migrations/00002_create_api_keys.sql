-- +goose Up
CREATE TABLE api_keys (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    prefix VARCHAR(20) NOT NULL,
    suffix VARCHAR(4) NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE api_keys;
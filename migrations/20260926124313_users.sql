-- +goose Up
CREATE TABLE users (
    user_id        VARCHAR(50) PRIMARY KEY,
    user_name      VARCHAR(50) NOT NULL,
    prompt_score   INTEGER      NOT NULL DEFAULT 0,
    response_score INTEGER      NOT NULL DEFAULT 0,
    total_score    INTEGER      NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_created_at ON users (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS users;

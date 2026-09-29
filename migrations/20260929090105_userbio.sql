-- +goose Up
ALTER TABLE users
    ADD COLUMN google_id VARCHAR(100) NOT NULL,
    ADD COLUMN bio VARCHAR(100),
    ADD COLUMN links JSONB;

ALTER TABLE users
    ADD CONSTRAINT users_google_id_key UNIQUE (google_id);

-- +goose Down
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_google_id_key;

ALTER TABLE users
    DROP COLUMN IF EXISTS links,
    DROP COLUMN IF EXISTS bio,
    DROP COLUMN IF EXISTS google_id;
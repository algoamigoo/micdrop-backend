-- +goose Up
CREATE TABLE follows (
    follower_id VARCHAR(50) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    followee_id VARCHAR(50) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, followee_id),
    CONSTRAINT follows_no_self CHECK (follower_id <> followee_id)
);

-- Counting followers and listing a user's following both key on followee_id.
CREATE INDEX idx_follows_followee ON follows (followee_id);

-- +goose Down
DROP TABLE IF EXISTS follows;
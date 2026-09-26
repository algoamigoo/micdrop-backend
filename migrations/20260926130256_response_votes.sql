-- +goose Up
CREATE TABLE response_votes (
    user_id     VARCHAR(50) NOT NULL REFERENCES users(user_id)         ON DELETE CASCADE,
    response_id BIGINT      NOT NULL REFERENCES responses(response_id)  ON DELETE CASCADE,
    vote_type   VARCHAR(10) NOT NULL CHECK (vote_type IN ('upvote', 'downvote')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, response_id)
);

-- +goose Down
DROP TABLE IF EXISTS response_votes;

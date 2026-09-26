-- +goose Up
CREATE TABLE prompt_votes (
    user_id    VARCHAR(50) NOT NULL REFERENCES users(user_id)   ON DELETE CASCADE,
    post_id    BIGINT      NOT NULL REFERENCES prompts(post_id) ON DELETE CASCADE,
    vote_type  VARCHAR(10) NOT NULL CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, post_id)
);

-- +goose Down
DROP TABLE IF EXISTS prompt_votes;

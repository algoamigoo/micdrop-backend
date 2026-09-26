-- +goose Up
CREATE TABLE responses (
    response_id      BIGSERIAL    PRIMARY KEY,
    post_id          BIGINT       NOT NULL REFERENCES prompts(post_id) ON DELETE CASCADE,
    user_id          VARCHAR(50)  NOT NULL REFERENCES users(user_id)  ON DELETE CASCADE,
    body             VARCHAR(280) NOT NULL,
    response_upvotes INTEGER      NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_responses_post_id    ON responses (post_id);
CREATE INDEX idx_responses_user_id   ON responses (user_id);
CREATE INDEX idx_responses_created_at ON responses (created_at DESC);
CREATE INDEX idx_responses_upvotes    ON responses (response_upvotes DESC);

-- +goose Down
DROP TABLE IF EXISTS responses;
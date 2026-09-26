-- +goose Up
CREATE TABLE prompts (
    post_id         BIGSERIAL    PRIMARY KEY,
    user_id         VARCHAR(50)  NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    body            VARCHAR(280) NOT NULL,
    prompt_upvotes  INTEGER      NOT NULL DEFAULT 0,
    response_count  INTEGER      NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prompts_created_at ON prompts (created_at DESC);
CREATE INDEX idx_prompts_upvotes    ON prompts (prompt_upvotes DESC);
CREATE INDEX idx_prompts_user_id    ON prompts (user_id);

-- +goose Down
DROP TABLE IF EXISTS prompts;
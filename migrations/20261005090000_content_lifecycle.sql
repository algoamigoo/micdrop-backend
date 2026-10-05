-- +goose Up
-- Soft delete for user-authored content. See docs/edit-delete-decisions.md.
ALTER TABLE prompts   ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE responses ADD COLUMN deleted_at TIMESTAMPTZ;

-- Every read filters deleted_at IS NULL, so partial indexes only carry live rows.
CREATE INDEX idx_prompts_live_created ON prompts (created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_responses_live_post ON responses (post_id, response_upvotes DESC, created_at ASC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_responses_live_post;
DROP INDEX IF EXISTS idx_prompts_live_created;

ALTER TABLE responses DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE prompts   DROP COLUMN IF EXISTS deleted_at;
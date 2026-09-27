package repository

import (
	"context"

	"github.com/algoamigoo/micdrop/internal/models"
)

// CreateResponse inserts a new response and bumps the parent prompt's response_count.
// It uses a transaction so both operations succeed or fail together.
func (r *Repository) CreateResponse(ctx context.Context, postID int64, userID, body string) (*models.Response, error) {
	// Start the transaction
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	// Rollback is safe to call even if the commit succeeds later.
	// It just ensures cleanup if we return early due to an error.
	defer tx.Rollback()

	// 1. Insert the response
	insertQuery := `
        INSERT INTO responses (post_id, user_id, body)
        VALUES ($1, $2, $3)
        RETURNING response_id, post_id, user_id, body, response_upvotes, created_at, updated_at;
    `
	var resp models.Response
	err = tx.QueryRowxContext(ctx, insertQuery, postID, userID, body).StructScan(&resp)
	if err != nil {
		// A foreign key violation here means the post_id doesn't exist
		return nil, err
	}

	// 2. Bump the response_count on the parent prompt
	updateQuery := `UPDATE prompts SET response_count = response_count + 1 WHERE post_id = $1;`
	_, err = tx.ExecContext(ctx, updateQuery, postID)
	if err != nil {
		return nil, err
	}

	// 3. Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListResponsesForPrompt fetches all responses for a specific prompt.
func (r *Repository) ListResponsesForPrompt(ctx context.Context, postID int64, limit, offset int) ([]models.Response, error) {
	// Order by upvotes descending (best punchlines first), then oldest first
	query := `
        SELECT r.response_id, r.post_id, r.user_id, u.user_name, r.body, r.response_upvotes, r.created_at, r.updated_at
        FROM responses r
        JOIN users u ON r.user_id = u.user_id
        WHERE r.post_id = $1
        ORDER BY r.response_upvotes DESC, r.created_at ASC
        LIMIT $2 OFFSET $3;
    `

	var responses []models.Response
	err := r.db.SelectContext(ctx, &responses, query, postID, limit, offset)
	if err != nil {
		return nil, err
	}

	return responses, nil
}

// GetLeaderboard fetches the top users based on total_score.
func (r *Repository) GetLeaderboard(ctx context.Context, limit int) ([]models.User, error) {
	query := `
        SELECT user_id, user_name, prompt_score, response_score, total_score, created_at, updated_at
        FROM users
        ORDER BY total_score DESC
        LIMIT $1;
    `

	var users []models.User
	err := r.db.SelectContext(ctx, &users, query, limit)
	if err != nil {
		return nil, err
	}

	return users, nil
}

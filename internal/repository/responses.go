package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreateResponse inserts a new response and bumps the parent prompt's response_count.
func (r *Repository) CreateResponse(ctx context.Context, postID int64, userID, body string) (*models.Response, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("repository.CreateResponse begin tx: %w", err)
	}
	defer tx.Rollback()

	insertQuery := `
		INSERT INTO responses (post_id, user_id, body)
		VALUES ($1, $2, $3)
		RETURNING response_id, post_id, user_id, body, response_upvotes, created_at, updated_at;
	`
	var resp models.Response
	err = tx.QueryRowxContext(ctx, insertQuery, postID, userID, body).StructScan(&resp)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			// Check which foreign key actually failed
			if pgErr.ConstraintName == "responses_user_id_fkey" {
				return nil, ErrUserNotFound
			}
			return nil, ErrPromptNotFound
		}
		return nil, fmt.Errorf("repository.CreateResponse insert: %w", err)
	}

	updateQuery := `UPDATE prompts SET response_count = response_count + 1 WHERE post_id = $1;`
	_, err = tx.ExecContext(ctx, updateQuery, postID)
	if err != nil {
		return nil, fmt.Errorf("repository.CreateResponse update count: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.CreateResponse commit: %w", err)
	}

	return &resp, nil
}

// ListResponsesForPrompt fetches all responses for a specific prompt.
func (r *Repository) ListResponsesForPrompt(ctx context.Context, postID int64, limit, offset int) ([]models.Response, error) {
	query := `
		SELECT response_id, post_id, user_id, body, response_upvotes, created_at, updated_at
		FROM responses
		WHERE post_id = $1
		ORDER BY response_upvotes DESC, created_at ASC
		LIMIT $2 OFFSET $3;
	`

	var responses []models.Response
	err := r.db.SelectContext(ctx, &responses, query, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("repository.ListResponsesForPrompt: %w", err)
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
		return nil, fmt.Errorf("repository.GetLeaderboard: %w", err)
	}

	return users, nil
}

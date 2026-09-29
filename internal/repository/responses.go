package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreateResponse inserts a response, bumps the parent's response_count,
// and records the author's auto-upvote (counter starts at 1, no karma for self-vote).
func (r *Repository) CreateResponse(ctx context.Context, postID int64, userID, body string) (*models.Response, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("repository.CreateResponse begin tx: %w", err)
	}
	defer tx.Rollback()

	var responseID int64
	err = tx.QueryRowxContext(ctx, `
		INSERT INTO responses (post_id, user_id, body, response_upvotes)
		VALUES ($1, $2, $3, 1)
		RETURNING response_id;`, postID, userID, body).Scan(&responseID)
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

	if _, err := tx.ExecContext(ctx, `
        INSERT INTO response_votes (user_id, response_id, vote_type)
        VALUES ($1, $2, 'upvote');`, userID, responseID); err != nil {
		return nil, fmt.Errorf("repository.CreateResponse self-vote: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE prompts SET response_count = response_count + 1 WHERE post_id = $1;`, postID); err != nil {
		return nil, fmt.Errorf("repository.CreateResponse update count: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.CreateResponse commit: %w", err)
	}

	return r.GetResponseByID(ctx, responseID, userID)
}

// GetResponseByID fetches a single response with the viewer's vote.
func (r *Repository) GetResponseByID(ctx context.Context, responseID int64, viewerID string) (*models.Response, error) {
	query := `
		SELECT r.response_id, r.post_id, r.user_id, r.body, r.response_upvotes,
		    rv.vote_type AS viewer_vote,
		    r.created_at, r.updated_at
		FROM responses r
		LEFT JOIN response_votes rv ON rv.response_id = r.response_id AND rv.user_id = $2
		WHERE r.response_id = $1;
	`

	var resp models.Response
	if err := r.db.GetContext(ctx, &resp, query, responseID, viewerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrResponseNotFound
		}
		return nil, fmt.Errorf("repository.GetResponseByID: %w", err)
	}

	return &resp, nil
}

// ListResponsesForPrompt fetches responses for a prompt with viewer votes.
func (r *Repository) ListResponsesForPrompt(ctx context.Context, postID int64, limit, offset int, viewerID string) ([]models.Response, error) {
	query := `
		SELECT r.response_id, r.post_id, r.user_id, r.body, r.response_upvotes,
		    rv.vote_type AS viewer_vote,
		    r.created_at, r.updated_at
		FROM responses r
		LEFT JOIN response_votes rv ON rv.response_id = r.response_id AND rv.user_id = $4
		WHERE r.post_id = $1
		ORDER BY r.response_upvotes DESC, r.created_at ASC
		LIMIT $2 OFFSET $3;
	`

	var responses []models.Response
	err := r.db.SelectContext(ctx, &responses, query, postID, limit, offset, viewerID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListResponsesForPrompt: %w", err)
	}
	if responses == nil {
		responses = []models.Response{}
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

// ListResponsesByUser fetches responses by author with viewer votes.
func (r *Repository) ListResponsesByUser(ctx context.Context, userID string, limit, offset int, viewerID string) ([]models.Response, error) {
	query := `
        SELECT r.response_id, r.post_id, r.user_id, r.body, r.response_upvotes,
            rv.vote_type AS viewer_vote,
            r.created_at, r.updated_at
        FROM responses r
        LEFT JOIN response_votes rv ON rv.response_id = r.response_id AND rv.user_id = $4
        WHERE r.user_id = $1
        ORDER BY r.created_at DESC
        LIMIT $2 OFFSET $3;
    `
	var responses []models.Response
	err := r.db.SelectContext(ctx, &responses, query, userID, limit, offset, viewerID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListResponsesByUser: %w", err)
	}
	if responses == nil {
		responses = []models.Response{}
	}
	return responses, nil
}

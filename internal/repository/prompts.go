package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreatePrompt inserts a new prompt plus the author's auto-upvote (counter starts at 1).
// The self-vote counts toward the displayed count but not toward author karma.
func (r *Repository) CreatePrompt(ctx context.Context, userID, body string) (*models.Prompt, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository.CreatePrompt begin tx: %w", err)
	}
	defer tx.Rollback()

	var postID int64
	err = tx.QueryRowxContext(ctx, `
        INSERT INTO prompts (user_id, body, prompt_upvotes)
        VALUES ($1, $2, 1)
        RETURNING post_id;`, userID, body).Scan(&postID)
	if err != nil {
		// Foreign-key violation: the author does not exist.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("repository.CreatePrompt insert: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
        INSERT INTO prompt_votes (user_id, post_id, vote_type)
        VALUES ($1, $2, 'upvote');`, userID, postID); err != nil {
		return nil, fmt.Errorf("repository.CreatePrompt self-vote: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.CreatePrompt commit: %w", err)
	}

	return r.GetPromptByID(ctx, postID, userID)
}

// GetPromptByID fetches a single prompt with the viewer's vote (empty viewerID = anonymous).
func (r *Repository) GetPromptByID(ctx context.Context, postID int64, viewerID string) (*models.Prompt, error) {
	query := `
        SELECT p.post_id, p.user_id, p.body, p.prompt_upvotes, p.response_count,
            pv.vote_type AS viewer_vote,
            p.created_at, p.updated_at
        FROM prompts p
        LEFT JOIN prompt_votes pv ON pv.post_id = p.post_id AND pv.user_id = $2
        WHERE p.post_id = $1;
    `

	var prompt models.Prompt
	err := r.db.GetContext(ctx, &prompt, query, postID, viewerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromptNotFound
		}
		return nil, fmt.Errorf("repository.GetPromptByID: %w", err)
	}

	return &prompt, nil
}

// ListPrompts fetches prompts with viewer votes. Supported sorts: "newest", "top".
func (r *Repository) ListPrompts(ctx context.Context, sort string, limit, offset int, viewerID string) ([]models.Prompt, error) {
	orderBy := "p.created_at DESC" // Default to newest
	if sort == "top" {
		orderBy = "p.prompt_upvotes DESC, p.created_at DESC"
	}

	// Sprintf is used ONLY for the ORDER BY clause — it cannot be a bind parameter.
	query := fmt.Sprintf(`
        SELECT p.post_id, p.user_id, p.body, p.prompt_upvotes, p.response_count,
            pv.vote_type AS viewer_vote,
            p.created_at, p.updated_at
        FROM prompts p
        LEFT JOIN prompt_votes pv ON pv.post_id = p.post_id AND pv.user_id = $3
        ORDER BY %s
        LIMIT $1 OFFSET $2;`, orderBy)

	var prompts []models.Prompt
	err := r.db.SelectContext(ctx, &prompts, query, limit, offset, viewerID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListPrompts: %w", err)
	}
	if prompts == nil {
		prompts = []models.Prompt{}
	}

	return prompts, nil
}

// ListPromptsByUser fetches prompts by author with viewer votes.
func (r *Repository) ListPromptsByUser(ctx context.Context, userID string, limit, offset int, viewerID string) ([]models.Prompt, error) {
	query := `
        SELECT p.post_id, p.user_id, p.body, p.prompt_upvotes, p.response_count,
            pv.vote_type AS viewer_vote,
            p.created_at, p.updated_at
        FROM prompts p
        LEFT JOIN prompt_votes pv ON pv.post_id = p.post_id AND pv.user_id = $4
        WHERE p.user_id = $1
        ORDER BY p.created_at DESC
        LIMIT $2 OFFSET $3;
    `
	var prompts []models.Prompt
	err := r.db.SelectContext(ctx, &prompts, query, userID, limit, offset, viewerID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListPromptsByUser: %w", err)
	}
	if prompts == nil {
		prompts = []models.Prompt{}
	}
	return prompts, nil
}

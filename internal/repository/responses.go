package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
)

// responseColumns is the canonical SELECT list for responses rows, without the
// viewer's vote. Must match the field order of models.Response.
const responseColumns = `
    r.response_id, r.post_id, r.user_id, r.body, r.response_upvotes,
    (r.updated_at > r.created_at) AS edited,
    r.created_at, r.updated_at
`

// responseColumnsWithViewer adds the viewer's own vote via LEFT JOIN.
const responseColumnsWithViewer = responseColumns + `, rv.vote_type AS viewer_vote`

// CreateResponse inserts a response, bumps the parent's response_count,
// and records the author's auto-upvote (counter starts at 1, no karma for self-vote).
func (r *Repository) CreateResponse(ctx context.Context, postID int64, userID, body string) (*models.Response, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
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

// UpdateResponse edits the body of a response authored by userID.
//
// Returns ErrResponseNotFound if the response does not exist or is deleted, and
// ErrNotAuthor if userID is not the author.
func (r *Repository) UpdateResponse(ctx context.Context, responseID int64, userID, body string) (*models.Response, error) {
	var authorID string
	err := r.db.GetContext(ctx, &authorID,
		`SELECT user_id FROM responses WHERE response_id = $1 AND deleted_at IS NULL;`, responseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrResponseNotFound
		}
		return nil, fmt.Errorf("repository.UpdateResponse lookup: %w", err)
	}
	if authorID != userID {
		return nil, ErrNotAuthor
	}

	var resp models.Response
	query := `
        UPDATE responses SET body = $2, updated_at = NOW()
        WHERE response_id = $1 AND deleted_at IS NULL
        RETURNING response_id, post_id, user_id, body, response_upvotes,
                  (updated_at > created_at) AS edited, created_at, updated_at;`
	if err := r.db.QueryRowxContext(ctx, query, responseID, body).StructScan(&resp); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrResponseNotFound
		}
		return nil, fmt.Errorf("repository.UpdateResponse: %w", err)
	}
	return r.GetResponseByID(ctx, responseID, userID)
}

// DeleteResponse soft-deletes a response and decrements its prompt's response_count.
// Scores and karma are left alone. See docs/edit-delete-decisions.md.
func (r *Repository) DeleteResponse(ctx context.Context, responseID int64, userID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository.DeleteResponse begin tx: %w", err)
	}
	defer tx.Rollback()

	var authorID string
	var postID int64
	err = tx.QueryRowxContext(ctx,
		`SELECT user_id, post_id FROM responses WHERE response_id = $1 AND deleted_at IS NULL FOR UPDATE;`,
		responseID).Scan(&authorID, &postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrResponseNotFound
		}
		return fmt.Errorf("repository.DeleteResponse lock: %w", err)
	}
	if authorID != userID {
		return ErrNotAuthor
	}

	// Scores and karma are untouched: the upvotes were earned.
	if _, err := tx.ExecContext(ctx,
		`UPDATE responses SET deleted_at = NOW() WHERE response_id = $1;`, responseID); err != nil {
		return fmt.Errorf("repository.DeleteResponse delete response: %w", err)
	}

	// response_count is not a score, it counts visible children, so keep it honest.
	// The parent may itself be deleted, in which case its count no longer matters.
	if _, err := tx.ExecContext(ctx,
		`UPDATE prompts SET response_count = GREATEST(response_count - 1, 0)
         WHERE post_id = $1 AND deleted_at IS NULL;`, postID); err != nil {
		return fmt.Errorf("repository.DeleteResponse decrement count: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository.DeleteResponse commit: %w", err)
	}
	return nil
}

// GetResponseByID fetches a single response with the viewer's vote.
func (r *Repository) GetResponseByID(ctx context.Context, responseID int64, viewerID string) (*models.Response, error) {
	query := `
		SELECT ` + responseColumnsWithViewer + `
		FROM responses r
		LEFT JOIN response_votes rv ON rv.response_id = r.response_id AND rv.user_id = $2
		WHERE r.response_id = $1 AND r.deleted_at IS NULL;
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
		SELECT ` + responseColumnsWithViewer + `
		FROM responses r
		LEFT JOIN response_votes rv ON rv.response_id = r.response_id AND rv.user_id = $4
		WHERE r.post_id = $1 AND r.deleted_at IS NULL
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
        SELECT ` + responseColumnsWithViewer + `
        FROM responses r
        LEFT JOIN response_votes rv ON rv.response_id = r.response_id AND rv.user_id = $4
        WHERE r.user_id = $1 AND r.deleted_at IS NULL
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

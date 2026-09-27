package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/algoamigoo/micdrop/internal/models"
)

// CreatePrompt inserts a new prompt into the database.
func (r *Repository) CreatePrompt(ctx context.Context, userID, body string) (*models.Prompt, error) {
	query := `
        INSERT INTO prompts (user_id, body)
        VALUES ($1, $2)
        RETURNING post_id, user_id, body, prompt_upvotes, response_count, created_at, updated_at;
    `

	var prompt models.Prompt
	err := r.db.QueryRowxContext(ctx, query, userID, body).StructScan(&prompt)
	if err != nil {
		return nil, fmt.Errorf("repository.CreatePrompt: %w", err)
	}

	return &prompt, nil
}

// GetPromptByID fetches a single prompt by its ID.
func (r *Repository) GetPromptByID(ctx context.Context, postID int64) (*models.Prompt, error) {
	query := `
        SELECT post_id, user_id, body, prompt_upvotes, response_count, created_at, updated_at
        FROM prompts
        WHERE post_id = $1;
    `

	var prompt models.Prompt
	err := r.db.GetContext(ctx, &prompt, query, postID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromptNotFound
		}
		return nil, fmt.Errorf("repository.GetPromptByID: %w", err)
	}

	return &prompt, nil
}

// ListPrompts fetches a list of prompts based on the specified sort order.
// Supported sorts: "newest", "top". (Defaults to newest)
func (r *Repository) ListPrompts(ctx context.Context, sort string, limit, offset int) ([]models.Prompt, error) {
	orderBy := "created_at DESC" // Default to newest
	if sort == "top" {
		orderBy = "prompt_upvotes DESC, created_at DESC"
	}

	// We use Sprintf ONLY for the ORDER BY clause, as you cannot parameterize it.
	query := `
        SELECT post_id, user_id, body, prompt_upvotes, response_count, created_at, updated_at
        FROM prompts
        ORDER BY %s
        LIMIT $1 OFFSET $2;
    `

	finalQuery := fmt.Sprintf(query, orderBy)

	var prompts []models.Prompt
	err := r.db.SelectContext(ctx, &prompts, finalQuery, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("repository.ListPrompts: %w", err)
	}

	return prompts, nil
}

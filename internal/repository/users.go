package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/jmoiron/sqlx"
)

// Repository provides access to the database.
type Repository struct {
	db *sqlx.DB
}

// New creates a new Repository instance.
func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// GetOrCreateUser creates a user if they don't exist, or returns the existing one.
// If userName is empty, it defaults to the userID (enforced in Go).
func (r *Repository) GetOrCreateUser(ctx context.Context, userID, userName string) (*models.User, error) {
	if userName == "" {
		userName = userID
	}

	query := `
        INSERT INTO users (user_id, user_name)
        VALUES ($1, $2)
        ON CONFLICT (user_id) DO UPDATE 
        SET user_name = users.user_name
        RETURNING user_id, user_name, prompt_score, response_score, total_score, created_at, updated_at;
    `

	var user models.User
	err := r.db.QueryRowxContext(ctx, query, userID, userName).StructScan(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByID fetches a user by their immutable user_id.
func (r *Repository) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	query := `
        SELECT user_id, user_name, prompt_score, response_score, total_score, created_at, updated_at
        FROM users
        WHERE user_id = $1;
    `

	var user models.User
	err := r.db.GetContext(ctx, &user, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

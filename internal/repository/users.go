package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
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

// userColumns is the canonical SELECT list for users rows. It must match
// the field order of models.User for sqlx.StructScan / GetContext.
const userColumns = `
    user_id, user_name, google_id, bio, links,
    prompt_score, response_score, total_score, created_at, updated_at
`

// GetUserByGoogleID fetches a user by the raw Google subject id.
// Returns ErrUserNotFound when no user is linked to that Google account.
func (r *Repository) GetUserByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE google_id = $1;`

	var user models.User
	if err := r.db.GetContext(ctx, &user, query, googleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("repository.GetUserByGoogleID: %w", err)
	}
	return &user, nil
}

// CreateUser inserts a brand-new user with a chosen user_id (username).
// user_name defaults to user_id; the Google account is linked via google_id.
//
// Returns:
//   - ErrUserIDTaken   on users_pkey violation (username already exists)
//   - ErrGoogleIDTaken on users_google_id_key violation (Google account
//     already linked)
func (r *Repository) CreateUser(ctx context.Context, userID, googleID string) (*models.User, error) {
	query := `
        INSERT INTO users (user_id, user_name, google_id)
        VALUES ($1, $1, $2)
        RETURNING ` + userColumns + `;`

	var user models.User
	if err := r.db.QueryRowxContext(ctx, query, userID, googleID).StructScan(&user); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_pkey":
				return nil, ErrUserIDTaken
			case "users_google_id_key":
				return nil, ErrGoogleIDTaken
			}
		}
		return nil, fmt.Errorf("repository.CreateUser: %w", err)
	}
	return &user, nil
}

// GetUserByID fetches a user by their immutable user_id.
// Returns ErrUserNotFound if no such user exists.
func (r *Repository) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE user_id = $1;`

	var user models.User
	if err := r.db.GetContext(ctx, &user, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("repository.GetUserByID: %w", err)
	}
	return &user, nil
}

// UpdateProfileInput carries the mutable profile fields.
//
// A nil field means "leave unchanged" (handled via COALESCE in SQL).
//   - UserName: nil leaves the display name alone. Non-nil sets it.
//   - Bio:      nil leaves the bio alone. Non-nil sets it (empty string
//     is a valid bio value, not a clear).
//   - Links:    nil leaves links alone. A pointer to a (possibly empty)
//     slice sets the column — an empty slice clears it.
type UpdateProfileInput struct {
	UserName *string
	Bio      *string
	Links    *models.Links
}

// UpdateProfile partially updates a user's profile and returns the fresh row.
// Returns ErrUserNotFound if the user does not exist.
func (r *Repository) UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*models.User, error) {
	// Dereference Links so database/sql sees either a nil interface
	// (→ SQL NULL → COALESCE keeps existing value) or a driver.Valuer
	// (→ JSONB, possibly empty array → sets column).
	var linksArg any
	if input.Links != nil {
		linksArg = *input.Links
	}

	query := `
        UPDATE users
        SET user_name  = COALESCE($2, user_name),
            bio        = COALESCE($3, bio),
            links      = COALESCE($4, links),
            updated_at = NOW()
        WHERE user_id = $1
        RETURNING ` + userColumns + `;`

	var user models.User
	err := r.db.QueryRowxContext(ctx, query, userID, input.UserName, input.Bio, linksArg).StructScan(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("repository.UpdateProfile: %w", err)
	}
	return &user, nil
}

// GetUserStats returns prompt/response counts for a user.
// Returns ErrUserNotFound if the user does not exist.
func (r *Repository) GetUserStats(ctx context.Context, userID string) (*models.UserStats, error) {
	query := `
        SELECT
            (SELECT COUNT(*) FROM prompts   p WHERE p.user_id = u.user_id AND p.deleted_at IS NULL) AS prompt_count,
            (SELECT COUNT(*) FROM responses r WHERE r.user_id = u.user_id AND r.deleted_at IS NULL) AS response_count
        FROM users u
        WHERE u.user_id = $1;`

	var stats models.UserStats
	if err := r.db.GetContext(ctx, &stats, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("repository.GetUserStats: %w", err)
	}
	return &stats, nil
}

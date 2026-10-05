package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/algoamigoo/micdrop/internal/models"
)

// Follow makes followerID follow followeeID. Idempotent: following twice is a no-op.
//
// Returns ErrSelfFollow when the ids match and ErrUserNotFound when the target does
// not exist (a FK violation is not useful to the caller).
func (r *Repository) Follow(ctx context.Context, followerID, followeeID string) error {
	if followerID == followeeID {
		return ErrSelfFollow
	}

	var exists bool
	if err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS (SELECT 1 FROM users WHERE user_id = $1);`, followeeID); err != nil {
		return fmt.Errorf("repository.Follow lookup: %w", err)
	}
	if !exists {
		return ErrUserNotFound
	}

	query := `
        INSERT INTO follows (follower_id, followee_id)
        VALUES ($1, $2)
        ON CONFLICT (follower_id, followee_id) DO NOTHING;`
	if _, err := r.db.ExecContext(ctx, query, followerID, followeeID); err != nil {
		return fmt.Errorf("repository.Follow: %w", err)
	}
	return nil
}

// Unfollow removes a follow edge. Idempotent: unfollowing someone you do not follow
// succeeds, so the client can retry without special-casing.
func (r *Repository) Unfollow(ctx context.Context, followerID, followeeID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2;`,
		followerID, followeeID)
	if err != nil {
		return fmt.Errorf("repository.Unfollow: %w", err)
	}
	return nil
}

// GetFollowCounts returns follower/following counts for userID and whether the viewer
// follows them. An empty viewerID means anonymous, so IsFollowing is false.
func (r *Repository) GetFollowCounts(ctx context.Context, userID, viewerID string) (*models.FollowCounts, error) {
	query := `
        SELECT
            (SELECT COUNT(*) FROM follows WHERE followee_id = $1) AS followers_count,
            (SELECT COUNT(*) FROM follows WHERE follower_id = $1) AS following_count,
            EXISTS (SELECT 1 FROM follows WHERE follower_id = $2 AND followee_id = $1) AS is_following;`

	var counts models.FollowCounts
	if err := r.db.GetContext(ctx, &counts, query, userID, viewerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("repository.GetFollowCounts: %w", err)
	}
	return &counts, nil
}

// followUserColumns is userColumns with the users alias applied, for queries that
// reach users through the follows table. The field order must match models.User.
const followUserColumns = `
    u.user_id, u.user_name, u.google_id, u.bio, u.links,
    u.prompt_score, u.response_score, u.total_score, u.created_at, u.updated_at
`

// ListFollowers returns the users following userID, newest follow first.
// Returns ErrUserNotFound if userID does not exist.
func (r *Repository) ListFollowers(ctx context.Context, userID string, limit, offset int) ([]models.User, error) {
	query := `
        SELECT ` + followUserColumns + `
        FROM follows f
        JOIN users u ON u.user_id = f.follower_id
        WHERE f.followee_id = $1
        ORDER BY f.created_at DESC, f.follower_id
        LIMIT $2 OFFSET $3;`

	users := []models.User{}
	if err := r.db.SelectContext(ctx, &users, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("repository.ListFollowers: %w", err)
	}
	return users, nil
}

// ListFollowing returns the users userID follows, most recent first.
func (r *Repository) ListFollowing(ctx context.Context, userID string, limit, offset int) ([]models.User, error) {
	query := `
        SELECT ` + followUserColumns + `
        FROM follows f
        JOIN users u ON u.user_id = f.followee_id
        WHERE f.follower_id = $1
        ORDER BY f.created_at DESC, f.followee_id
        LIMIT $2 OFFSET $3;`

	users := []models.User{}
	if err := r.db.SelectContext(ctx, &users, query, userID, limit, offset); err != nil {
		return nil, fmt.Errorf("repository.ListFollowing: %w", err)
	}
	return users, nil
}

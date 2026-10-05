package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/algoamigoo/micdrop/internal/models"
)

// Vote values
const (
	VoteUp   = "upvote"
	VoteDown = "downvote"
	VoteNone = "none"
)

var ErrInvalidVoteType = errors.New("invalid vote type: must be 'upvote', 'downvote' or 'none'")

func voteValue(vote string) (int, error) {
	switch vote {
	case VoteUp:
		return 1, nil
	case VoteDown:
		return -1, nil
	case VoteNone:
		return 0, nil
	default:
		return 0, ErrInvalidVoteType
	}
}

// SetPromptVote sets the voter's desired state idempotently.
// Same-direction clicks clear (handled by client sending "none"); flips and
// retries converge because the client sends the desired end state.
// Self-votes count toward the displayed score but not toward author karma.
func (r *Repository) SetPromptVote(ctx context.Context, voterID string, postID int64, vote string) (*models.Prompt, error) {
	newVal, err := voteValue(vote)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository.SetPromptVote begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Lock the parent row: serializes votes per item, real 404, author id.
	var authorID string
	err = tx.QueryRowxContext(ctx,
		`SELECT user_id FROM prompts WHERE post_id = $1 FOR UPDATE;`, postID).Scan(&authorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromptNotFound
		}
		return nil, fmt.Errorf("repository.SetPromptVote lock prompt: %w", err)
	}

	// 2. Read the old vote (or none).
	var oldVote *string
	err = tx.QueryRowxContext(ctx,
		`SELECT vote_type FROM prompt_votes WHERE user_id = $1 AND post_id = $2;`,
		voterID, postID).Scan(&oldVote)
	oldVal := 0
	if err == nil && oldVote != nil {
		if v, verr := voteValue(*oldVote); verr == nil {
			oldVal = v
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("repository.SetPromptVote read vote: %w", err)
	}

	delta := newVal - oldVal
	if delta != 0 {
		// 3. Delete or upsert the vote row.
		if vote == VoteNone {
			if _, err := tx.ExecContext(ctx,
				`DELETE FROM prompt_votes WHERE user_id = $1 AND post_id = $2;`,
				voterID, postID); err != nil {
				return nil, fmt.Errorf("repository.SetPromptVote delete vote: %w", err)
			}
		} else {
			if _, err := tx.ExecContext(ctx, `
                INSERT INTO prompt_votes (user_id, post_id, vote_type)
                VALUES ($1, $2, $3)
                ON CONFLICT (user_id, post_id) DO UPDATE SET vote_type = EXCLUDED.vote_type;`,
				voterID, postID, vote); err != nil {
				return nil, fmt.Errorf("repository.SetPromptVote upsert vote: %w", err)
			}
		}

		// 4. Apply delta to the counter.
		if _, err := tx.ExecContext(ctx,
			`UPDATE prompts SET prompt_upvotes = prompt_upvotes + $1 WHERE post_id = $2;`, delta, postID); err != nil {
			return nil, fmt.Errorf("repository.SetPromptVote update score: %w", err)
		}

		// 5. Karma unless self-vote.
		if voterID != authorID {
			if _, err := tx.ExecContext(ctx, `
                UPDATE users
                SET prompt_score = prompt_score + $1, total_score = total_score + $1
                WHERE user_id = $2;`, delta, authorID); err != nil {
				return nil, fmt.Errorf("repository.SetPromptVote update karma: %w", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.SetPromptVote commit: %w", err)
	}

	return r.GetPromptByID(ctx, postID, voterID)
}

// SetResponseVote is the response analogue of SetPromptVote.
func (r *Repository) SetResponseVote(ctx context.Context, voterID string, responseID int64, vote string) (*models.Response, error) {
	newVal, err := voteValue(vote)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository.SetResponseVote begin tx: %w", err)
	}
	defer tx.Rollback()

	var authorID string
	err = tx.QueryRowxContext(ctx,
		`SELECT user_id FROM responses WHERE response_id = $1 FOR UPDATE;`, responseID).Scan(&authorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrResponseNotFound
		}
		return nil, fmt.Errorf("repository.SetResponseVote lock response: %w", err)
	}

	var oldVote *string
	err = tx.QueryRowxContext(ctx,
		`SELECT vote_type FROM response_votes WHERE user_id = $1 AND response_id = $2;`,
		voterID, responseID).Scan(&oldVote)
	oldVal := 0
	if err == nil && oldVote != nil {
		if v, verr := voteValue(*oldVote); verr == nil {
			oldVal = v
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("repository.SetResponseVote read vote: %w", err)
	}

	delta := newVal - oldVal
	if delta != 0 {
		if vote == VoteNone {
			if _, err := tx.ExecContext(ctx,
				`DELETE FROM response_votes WHERE user_id = $1 AND response_id = $2;`,
				voterID, responseID); err != nil {
				return nil, fmt.Errorf("repository.SetResponseVote delete vote: %w", err)
			}
		} else {
			if _, err := tx.ExecContext(ctx, `
                INSERT INTO response_votes (user_id, response_id, vote_type)
                VALUES ($1, $2, $3)
                ON CONFLICT (user_id, response_id) DO UPDATE SET vote_type = EXCLUDED.vote_type;`,
				voterID, responseID, vote); err != nil {
				return nil, fmt.Errorf("repository.SetResponseVote upsert vote: %w", err)
			}
		}

		if _, err := tx.ExecContext(ctx,
			`UPDATE responses SET response_upvotes = response_upvotes + $1 WHERE response_id = $2;`, delta, responseID); err != nil {
			return nil, fmt.Errorf("repository.SetResponseVote update score: %w", err)
		}

		if voterID != authorID {
			if _, err := tx.ExecContext(ctx, `
                UPDATE users
                SET response_score = response_score + $1, total_score = total_score + $1
                WHERE user_id = $2;`, delta, authorID); err != nil {
				return nil, fmt.Errorf("repository.SetResponseVote update karma: %w", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.SetResponseVote commit: %w", err)
	}

	return r.GetResponseByID(ctx, responseID, voterID)
}

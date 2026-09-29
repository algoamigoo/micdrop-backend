package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/algoamigoo/micdrop/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
)

// Vote type constants
const (
	VoteTypeUp   = "upvote"
	VoteTypeDown = "downvote"
)

var ErrInvalidVoteType = errors.New("invalid vote type: must be 'upvote' or 'downvote'")

func getVoteValue(voteType string) (int, error) {
	switch voteType {
	case VoteTypeUp:
		return 1, nil
	case VoteTypeDown:
		return -1, nil
	default:
		return 0, ErrInvalidVoteType
	}
}

// VoteOnPrompt handles upvoting/downvoting a prompt.
func (r *Repository) VoteOnPrompt(ctx context.Context, userID string, postID int64, voteType string) (*models.Prompt, error) {

	voteValue, err := getVoteValue(voteType)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("repository.VoteOnPrompt begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Insert the vote record
	_, err = tx.ExecContext(ctx, `
        INSERT INTO prompt_votes (user_id, post_id, vote_type)
        VALUES ($1, $2, $3);`, userID, postID, voteType)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyVoted
		}
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, ErrPromptNotFound
		}
		return nil, fmt.Errorf("repository.VoteOnPrompt insert vote: %w", err)
	}

	// 2. Update the denormalized counter
	_, err = tx.ExecContext(ctx, `
        UPDATE prompts SET prompt_upvotes = prompt_upvotes + $1 WHERE post_id = $2;`, voteValue, postID)
	if err != nil {
		return nil, fmt.Errorf("repository.VoteOnPrompt update prompt count: %w", err)
	}

	// 3. Update the author's score
	_, err = tx.ExecContext(ctx, `
        UPDATE users 
        SET prompt_score = prompt_score + $1, total_score = total_score + $1 
        WHERE user_id = (SELECT user_id FROM prompts WHERE post_id = $2);`, voteValue, postID)
	if err != nil {
		return nil, fmt.Errorf("repository.VoteOnPrompt update score: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.VoteOnPrompt commit: %w", err)
	}

	return r.GetPromptByID(ctx, postID)
}

// VoteOnResponse handles upvoting/downvoting a response.
func (r *Repository) VoteOnResponse(ctx context.Context, userID string, responseID int64, voteType string) (*models.Response, error) {

	voteValue, err := getVoteValue(voteType)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("repository.VoteOnResponse begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Insert the vote record
	_, err = tx.ExecContext(ctx, `
        INSERT INTO response_votes (user_id, response_id, vote_type)
        VALUES ($1, $2, $3);`, userID, responseID, voteType)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyVoted
		}
		if errors.As(err, &pgErr) && pgErr.Code == "23503" { // Foreign key violation
			return nil, ErrResponseNotFound
		}
		return nil, fmt.Errorf("repository.VoteOnResponse insert vote: %w", err)
	}

	// 2. Update the denormalized counter
	_, err = tx.ExecContext(ctx, `
        UPDATE responses SET response_upvotes = response_upvotes + $1 WHERE response_id = $2;`, voteValue, responseID)
	if err != nil {
		return nil, fmt.Errorf("repository.VoteOnResponse update response count: %w", err)
	}

	// 3. Update the author's score
	_, err = tx.ExecContext(ctx, `
        UPDATE users 
        SET response_score = response_score + $1, total_score = total_score + $1 
        WHERE user_id = (SELECT user_id FROM responses WHERE response_id = $2);`, voteValue, responseID)
	if err != nil {
		return nil, fmt.Errorf("repository.VoteOnResponse update score: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.VoteOnResponse commit: %w", err)
	}

	// 4. Fetch and return the updated response
	var resp models.Response
	err = r.db.GetContext(ctx, &resp, `
        SELECT response_id, post_id, user_id, body, response_upvotes, created_at, updated_at
        FROM responses
        WHERE response_id = $1;`, responseID)
	if err != nil {
		return nil, fmt.Errorf("repository.VoteOnResponse fetch updated: %w", err)
	}

	return &resp, nil
}

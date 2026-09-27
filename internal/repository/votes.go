package repository

import (
	"context"
	"errors"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
)

// VoteOnPrompt handles upvoting/downvoting a prompt.
// voteValue should be 1 for an upvote, -1 for a downvote.
func (r *Repository) VoteOnPrompt(ctx context.Context, userID string, postID int64, voteValue int) (*models.Prompt, error) {
	// 1. Fetch the prompt to check for self-voting
	var authorID string
	err := r.db.GetContext(ctx, &authorID, `SELECT user_id FROM prompts WHERE post_id = $1;`, postID)
	if err != nil {
		return nil, ErrPromptNotFound
	}

	if authorID == userID {
		return nil, ErrSelfVote
	}

	// 2. Start the transaction
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 3. Insert the vote record
	insertVoteQuery := `
        INSERT INTO prompt_votes (user_id, post_id, vote_value)
        VALUES ($1, $2, $3);
    `
	_, err = tx.ExecContext(ctx, insertVoteQuery, userID, postID, voteValue)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// 23505 is the Postgres unique_violation error code
			return nil, ErrAlreadyVoted
		}
		return nil, err
	}

	// 4. Update the denormalized counter on the prompt
	updatePromptQuery := `
        UPDATE prompts SET prompt_upvotes = prompt_upvotes + $1 WHERE post_id = $2;
    `
	_, err = tx.ExecContext(ctx, updatePromptQuery, voteValue, postID)
	if err != nil {
		return nil, err
	}

	// 5. Update the author's karma
	updateUserKarmaQuery := `
        UPDATE users 
        SET prompt_score = prompt_score + $1, total_score = total_score + $1 
        WHERE user_id = $2;
    `
	_, err = tx.ExecContext(ctx, updateUserKarmaQuery, voteValue, authorID)
	if err != nil {
		return nil, err
	}

	// 6. Commit
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// 7. Return the updated prompt
	return r.GetPromptByID(ctx, postID)
}

// VoteOnResponse handles upvoting/downvoting a response.
// voteValue should be 1 for an upvote, -1 for a downvote.
func (r *Repository) VoteOnResponse(ctx context.Context, userID string, responseID int64, voteValue int) (*models.Response, error) {
	// 1. Fetch the response to check for self-voting and get the parent post_id
	var resp models.Response
	err := r.db.GetContext(ctx, &resp, `SELECT response_id, post_id, user_id FROM responses WHERE response_id = $1;`, responseID)
	if err != nil {
		return nil, ErrResponseNotFound
	}

	if resp.UserID == userID {
		return nil, ErrSelfVote
	}

	// 2. Start the transaction
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 3. Insert the vote record
	insertVoteQuery := `
        INSERT INTO response_votes (user_id, response_id, vote_value)
        VALUES ($1, $2, $3);
    `
	_, err = tx.ExecContext(ctx, insertVoteQuery, userID, responseID, voteValue)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyVoted
		}
		return nil, err
	}

	// 4. Update the denormalized counter on the response
	updateResponseQuery := `
        UPDATE responses SET response_upvotes = response_upvotes + $1 WHERE response_id = $2;
    `
	_, err = tx.ExecContext(ctx, updateResponseQuery, voteValue, responseID)
	if err != nil {
		return nil, err
	}

	// 5. Update the author's score
	updateUserScoreQuery := `
        UPDATE users 
        SET response_score = response_score + $1, total_score = total_score + $1 
        WHERE user_id = $2;
    `
	_, err = tx.ExecContext(ctx, updateUserScoreQuery, voteValue, resp.UserID)
	if err != nil {
		return nil, err
	}

	// 6. Commit
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Return a basic response struct, or we could write a GetResponseByID method.
	// For brevity here, we just update the struct we already fetched.
	resp.ResponseUpvotes += voteValue
	return &resp, nil
}

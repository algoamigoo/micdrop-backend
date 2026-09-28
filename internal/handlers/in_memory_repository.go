package handlers

import (
	"context"

	"github.com/algoamigoo/micdrop/internal/models"
)

// InMemoryRepository implements the Repository interface for unit testing.
type InMemoryRepository struct {
	User      *models.User
	Prompt    *models.Prompt
	Prompts   []models.Prompt
	Response  *models.Response
	Responses []models.Response

	// Err allows us to simulate database errors
	Err error

	// Track inputs for assertions if needed
	CreatePromptCalledWith   struct{ UserID, Body string }
	CreateResponseCalledWith struct {
		PostID int64
		UserID string
		Body   string
	}
	VoteOnPromptCalledWith struct {
		UserID   string
		PostID   int64
		VoteType string
	}
}

func (r *InMemoryRepository) GetOrCreateUser(ctx context.Context, userID, userName string) (*models.User, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.User, nil
}

func (r *InMemoryRepository) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.User, nil
}

func (r *InMemoryRepository) CreatePrompt(ctx context.Context, userID, body string) (*models.Prompt, error) {
	r.CreatePromptCalledWith.UserID = userID
	r.CreatePromptCalledWith.Body = body
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Prompt, nil
}

func (r *InMemoryRepository) GetPromptByID(ctx context.Context, postID int64) (*models.Prompt, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Prompt, nil
}

func (r *InMemoryRepository) ListPrompts(ctx context.Context, sort string, limit, offset int) ([]models.Prompt, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Prompts, nil
}

func (r *InMemoryRepository) ListPromptsByUser(ctx context.Context, userID string, limit, offset int) ([]models.Prompt, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Prompts, nil
}

func (r *InMemoryRepository) VoteOnPrompt(ctx context.Context, userID string, postID int64, voteType string) (*models.Prompt, error) {
	r.VoteOnPromptCalledWith.UserID = userID
	r.VoteOnPromptCalledWith.PostID = postID
	r.VoteOnPromptCalledWith.VoteType = voteType
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Prompt, nil
}

func (r *InMemoryRepository) CreateResponse(ctx context.Context, postID int64, userID, body string) (*models.Response, error) {
	r.CreateResponseCalledWith.PostID = postID
	r.CreateResponseCalledWith.UserID = userID
	r.CreateResponseCalledWith.Body = body
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Response, nil
}

func (r *InMemoryRepository) ListResponsesForPrompt(ctx context.Context, postID int64, limit, offset int) ([]models.Response, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Responses, nil
}

func (r *InMemoryRepository) ListResponsesByUser(ctx context.Context, userID string, limit, offset int) ([]models.Response, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Responses, nil
}

func (r *InMemoryRepository) VoteOnResponse(ctx context.Context, userID string, responseID int64, voteType string) (*models.Response, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Response, nil
}

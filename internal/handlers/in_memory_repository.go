package handlers

import (
	"context"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
)

// InMemoryRepository implements the Repository interface for unit testing.
type InMemoryRepository struct {
	User      *models.User
	Prompt    *models.Prompt
	Prompts   []models.Prompt
	Response  *models.Response
	Responses []models.Response
	Stats     *models.UserStats

	// Err allows us to simulate database errors.
	Err error

	// Track inputs for assertions.
	CreateUserCalledWith struct {
		UserID   string
		GoogleID string
	}
	CreatePromptCalledWith struct {
		UserID string
		Body   string
	}
	CreateResponseCalledWith struct {
		PostID int64
		UserID string
		Body   string
	}
	UpdateProfileCalledWith struct {
		UserID string
		Input  repository.UpdateProfileInput
	}
	VoteOnPromptCalledWith struct {
		UserID   string
		PostID   int64
		VoteType string
	}
}

func (r *InMemoryRepository) GetUserByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	if r.User == nil {
		return nil, repository.ErrUserNotFound
	}
	return r.User, nil
}

func (r *InMemoryRepository) CreateUser(ctx context.Context, userID, googleID string) (*models.User, error) {
	r.CreateUserCalledWith.UserID = userID
	r.CreateUserCalledWith.GoogleID = googleID
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

func (r *InMemoryRepository) GetUserStats(ctx context.Context, userID string) (*models.UserStats, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	if r.Stats != nil {
		return r.Stats, nil
	}
	return &models.UserStats{}, nil
}

func (r *InMemoryRepository) UpdateProfile(ctx context.Context, userID string, input repository.UpdateProfileInput) (*models.User, error) {
	r.UpdateProfileCalledWith.UserID = userID
	r.UpdateProfileCalledWith.Input = input
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

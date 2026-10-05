package handlers

import (
	"context"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
)

type listCall struct {
	Kind   string
	UserID string
	Limit  int
	Offset int
}

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
	UpdatePromptCalledWith struct {
		PostID int64
		UserID string
		Body   string
	}
	DeletePromptCalledWith struct {
		PostID int64
		UserID string
	}
	UpdateResponseCalledWith struct {
		ResponseID int64
		UserID     string
		Body       string
	}
	DeleteResponseCalledWith struct {
		ResponseID int64
		UserID     string
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
	FollowCalledWith struct {
		FollowerID string
		FolloweeID string
	}
	ListCalledWith          []listCall
	Follows                 *models.FollowCounts
	FollowerUsers           []models.User
	FollowingUsers          []models.User
	SetPromptVoteCalledWith struct {
		UserID   string
		PostID   int64
		VoteType string
	}
	SetResponseVoteCalledWith struct {
		UserID     string
		ResponseID int64
		VoteType   string
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

func (r *InMemoryRepository) GetFollowCounts(ctx context.Context, userID, viewerID string) (*models.FollowCounts, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	if r.Follows != nil {
		return r.Follows, nil
	}
	return &models.FollowCounts{}, nil
}

func (r *InMemoryRepository) ListFollowers(ctx context.Context, userID string, limit, offset int) ([]models.User, error) {
	r.ListCalledWith = append(r.ListCalledWith, listCall{Kind: "followers", UserID: userID, Limit: limit, Offset: offset})
	if r.Err != nil {
		return nil, r.Err
	}
	// The repository always returns a non-nil slice.
	if r.FollowerUsers == nil {
		return []models.User{}, nil
	}
	return r.FollowerUsers, nil
}

func (r *InMemoryRepository) ListFollowing(ctx context.Context, userID string, limit, offset int) ([]models.User, error) {
	r.ListCalledWith = append(r.ListCalledWith, listCall{Kind: "following", UserID: userID, Limit: limit, Offset: offset})
	if r.Err != nil {
		return nil, r.Err
	}
	if r.FollowingUsers == nil {
		return []models.User{}, nil
	}
	return r.FollowingUsers, nil
}

func (r *InMemoryRepository) Follow(ctx context.Context, followerID, followeeID string) error {
	r.FollowCalledWith.FollowerID = followerID
	r.FollowCalledWith.FolloweeID = followeeID
	return r.Err
}

func (r *InMemoryRepository) Unfollow(ctx context.Context, followerID, followeeID string) error {
	r.FollowCalledWith.FollowerID = followerID
	r.FollowCalledWith.FolloweeID = followeeID
	return r.Err
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

func (r *InMemoryRepository) UpdatePrompt(ctx context.Context, postID int64, userID, body string) (*models.Prompt, error) {
	r.UpdatePromptCalledWith.UserID = userID
	r.UpdatePromptCalledWith.PostID = postID
	r.UpdatePromptCalledWith.Body = body
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Prompt, nil
}

func (r *InMemoryRepository) DeletePrompt(ctx context.Context, postID int64, userID string) error {
	r.DeletePromptCalledWith.UserID = userID
	r.DeletePromptCalledWith.PostID = postID
	return r.Err
}

func (r *InMemoryRepository) GetPromptByID(ctx context.Context, postID int64, viewerID string) (*models.Prompt, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Prompt, nil
}

func (r *InMemoryRepository) ListPrompts(ctx context.Context, sort string, limit, offset int, viewerID string) ([]models.Prompt, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Prompts, nil
}

func (r *InMemoryRepository) ListPromptsByUser(ctx context.Context, userID string, limit, offset int, viewerID string) ([]models.Prompt, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Prompts, nil
}

func (r *InMemoryRepository) SetPromptVote(ctx context.Context, voterID string, postID int64, vote string) (*models.Prompt, error) {
	r.SetPromptVoteCalledWith.UserID = voterID
	r.SetPromptVoteCalledWith.PostID = postID
	r.SetPromptVoteCalledWith.VoteType = vote
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

func (r *InMemoryRepository) UpdateResponse(ctx context.Context, responseID int64, userID, body string) (*models.Response, error) {
	r.UpdateResponseCalledWith.ResponseID = responseID
	r.UpdateResponseCalledWith.UserID = userID
	r.UpdateResponseCalledWith.Body = body
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Response, nil
}

func (r *InMemoryRepository) DeleteResponse(ctx context.Context, responseID int64, userID string) error {
	r.DeleteResponseCalledWith.ResponseID = responseID
	r.DeleteResponseCalledWith.UserID = userID
	return r.Err
}

func (r *InMemoryRepository) ListResponsesForPrompt(ctx context.Context, postID int64, limit, offset int, viewerID string) ([]models.Response, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Responses, nil
}

func (r *InMemoryRepository) ListResponsesByUser(ctx context.Context, userID string, limit, offset int, viewerID string) ([]models.Response, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Responses, nil
}

func (r *InMemoryRepository) SetResponseVote(ctx context.Context, voterID string, responseID int64, vote string) (*models.Response, error) {
	r.SetResponseVoteCalledWith.UserID = voterID
	r.SetResponseVoteCalledWith.ResponseID = responseID
	r.SetResponseVoteCalledWith.VoteType = vote
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Response, nil
}

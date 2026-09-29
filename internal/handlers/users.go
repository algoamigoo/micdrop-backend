package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/algoamigoo/micdrop/internal/middleware"
	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
	"github.com/go-chi/chi/v5"
)

// Repository defines the data access methods required by the handlers.
type Repository interface {
	GetUserByGoogleID(ctx context.Context, googleID string) (*models.User, error)
	CreateUser(ctx context.Context, userID, googleID string) (*models.User, error)
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
	GetUserStats(ctx context.Context, userID string) (*models.UserStats, error)
	UpdateProfile(ctx context.Context, userID string, input repository.UpdateProfileInput) (*models.User, error)

	CreatePrompt(ctx context.Context, userID, body string) (*models.Prompt, error)
	GetPromptByID(ctx context.Context, postID int64) (*models.Prompt, error)
	ListPrompts(ctx context.Context, sort string, limit, offset int) ([]models.Prompt, error)
	ListPromptsByUser(ctx context.Context, userID string, limit, offset int) ([]models.Prompt, error)
	VoteOnPrompt(ctx context.Context, userID string, postID int64, voteType string) (*models.Prompt, error)

	CreateResponse(ctx context.Context, postID int64, userID, body string) (*models.Response, error)
	ListResponsesForPrompt(ctx context.Context, postID int64, limit, offset int) ([]models.Response, error)
	ListResponsesByUser(ctx context.Context, userID string, limit, offset int) ([]models.Response, error)
	VoteOnResponse(ctx context.Context, userID string, responseID int64, voteType string) (*models.Response, error)
}

type Handler struct {
	Repo Repository
}

func New(repo Repository) *Handler {
	return &Handler{Repo: repo}
}

// GetUser returns a user's public profile with aggregate stats.
// GET /api/v1/users/{userID}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "user_id is required in path")
		return
	}

	user, err := h.Repo.GetUserByID(r.Context(), userID)
	if err != nil {
		handleAppError(w, err)
		return
	}

	stats, err := h.Repo.GetUserStats(r.Context(), userID)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"user":  user,
		"stats": stats,
	})
}

var allowedLinkTypes = map[string]struct{}{
	"github": {}, "twitter": {}, "youtube": {}, "instagram": {}, "linkedin": {}, "website": {},
}

var validLinkURL = regexp.MustCompile(`^https?://\S+$`)

type updateMeRequest struct {
	UserName *string        `json:"user_name"`
	Bio      *string        `json:"bio"`
	Links    *[]models.Link `json:"links"`
}

// UpdateMe updates the signed-in user's display name, bio, and links.
// PATCH /api/v1/users/me (auth required)
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req updateMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := repository.UpdateProfileInput{}

	if req.UserName != nil {
		name := strings.TrimSpace(*req.UserName)
		if n := utf8.RuneCountInString(name); n < 1 || n > 50 {
			respondError(w, http.StatusBadRequest, "user_name must be 1-50 characters")
			return
		}
		input.UserName = &name
	}

	if req.Bio != nil {
		bio := strings.TrimSpace(*req.Bio)
		if utf8.RuneCountInString(bio) > 100 {
			respondError(w, http.StatusBadRequest, "bio must be at most 100 characters")
			return
		}
		input.Bio = &bio
	}

	if req.Links != nil {
		if len(*req.Links) > 3 {
			respondError(w, http.StatusBadRequest, "at most 3 links are allowed")
			return
		}
		links := make(models.Links, 0, len(*req.Links))
		for _, l := range *req.Links {
			t := strings.ToLower(strings.TrimSpace(l.Type))
			if _, ok := allowedLinkTypes[t]; !ok {
				handleAppError(w, repository.ErrInvalidLink)
				return
			}
			u := strings.TrimSpace(l.URL)
			if !validLinkURL.MatchString(u) {
				handleAppError(w, repository.ErrInvalidLink)
				return
			}
			links = append(links, models.Link{Type: t, URL: u})
		}
		input.Links = &links
	}

	user, err := h.Repo.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// ListUserPrompts returns paginated prompts authored by a user.
// GET /api/v1/users/{userID}/prompts
func (h *Handler) ListUserPrompts(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "user_id is required in path")
		return
	}

	limit := 10
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 50 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	prompts, err := h.Repo.ListPromptsByUser(r.Context(), userID, limit, offset)
	if err != nil {
		handleAppError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, prompts)
}

// ListUserResponses returns paginated responses authored by a user.
// GET /api/v1/users/{userID}/responses
func (h *Handler) ListUserResponses(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "user_id is required in path")
		return
	}

	limit := 10
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 50 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	responses, err := h.Repo.ListResponsesByUser(r.Context(), userID, limit, offset)
	if err != nil {
		handleAppError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, responses)
}

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/go-chi/chi/v5"
)

// Repository defines the data access methods required by the handlers.
type Repository interface {
	GetOrCreateUser(ctx context.Context, userID, userName string) (*models.User, error)
	GetUserByID(ctx context.Context, userID string) (*models.User, error)

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

type createUserRequest struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	user, err := h.Repo.GetOrCreateUser(r.Context(), req.UserID, req.UserName)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, user)
}

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

	respondJSON(w, http.StatusOK, user)
}

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

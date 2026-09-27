package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/algoamigoo/micdrop/internal/repository"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Repo *repository.Repository
}

func New(repo *repository.Repository) *Handler {
	return &Handler{Repo: repo}
}

type createUserRequest struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}

// CreateUser handles POST /api/v1/users
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

// GetUser handles GET /api/v1/users/{userID}
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

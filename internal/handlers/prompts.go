package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/algoamigoo/micdrop/internal/middleware"
	"github.com/go-chi/chi/v5"
)

type createPromptRequest struct {
	Body string `json:"body"`
}

func (h *Handler) CreatePrompt(w http.ResponseWriter, r *http.Request) {
	var req createPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	if req.Body == "" {
		respondError(w, http.StatusBadRequest, "body is required")
		return
	}

	prompt, err := h.Repo.CreatePrompt(r.Context(), userID, req.Body)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, prompt)
}

// GetPrompt handles GET /api/v1/prompts/{postID}
func (h *Handler) GetPrompt(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid post_id format")
		return
	}

	prompt, err := h.Repo.GetPromptByID(r.Context(), postID)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, prompt)
}

// ListPrompts handles GET /api/v1/prompts?sort=newest&limit=10&offset=0
func (h *Handler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	// Default values
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "newest"
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10 // Default limit
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 && val <= 50 {
			limit = val
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0 // Default offset
	if offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	prompts, err := h.Repo.ListPrompts(r.Context(), sort, limit, offset)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, prompts)
}

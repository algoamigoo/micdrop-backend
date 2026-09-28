package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/algoamigoo/micdrop/internal/middleware"
	"github.com/go-chi/chi/v5"
)

type createResponseRequest struct {
	Body string `json:"body"`
}

func (h *Handler) CreateResponse(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid post_id format")
		return
	}

	var req createResponseRequest
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

	resp, err := h.Repo.CreateResponse(r.Context(), postID, userID, req.Body)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, resp)
}

// ListResponses handles GET /api/v1/prompts/{postID}/responses
func (h *Handler) ListResponses(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid post_id format")
		return
	}

	// Pagination defaults
	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	responses, err := h.Repo.ListResponsesForPrompt(r.Context(), postID, limit, offset)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, responses)
}

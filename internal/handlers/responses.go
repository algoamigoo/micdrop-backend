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

	if msg := validateBody(req.Body); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
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

	limit, offset, ok := parsePagination(w, r, 20, 100)
	if !ok {
		return
	}

	responses, err := h.Repo.ListResponsesForPrompt(r.Context(), postID, limit, offset, middleware.GetUserIDFromContext(r.Context()))
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, responses)
}

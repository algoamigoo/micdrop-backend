package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/algoamigoo/micdrop/internal/middleware"
)

type createResponseRequest struct {
	Body string `json:"body"`
}

func (h *Handler) CreateResponse(w http.ResponseWriter, r *http.Request) {
	postID, ok := pathID(w, r, "postID", "post_id")
	if !ok {
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
	postID, ok := pathID(w, r, "postID", "post_id")
	if !ok {
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

// UpdateResponse handles PATCH /api/v1/responses/{responseID} (auth required, author only).
func (h *Handler) UpdateResponse(w http.ResponseWriter, r *http.Request) {
	responseID, ok := pathID(w, r, "responseID", "response_id")
	if !ok {
		return
	}

	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	body, ok := decodePostBody(w, r)
	if !ok {
		return
	}

	resp, err := h.Repo.UpdateResponse(r.Context(), responseID, userID, body)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// DeleteResponse handles DELETE /api/v1/responses/{responseID} (auth required, author only).
// Soft-deletes the response and decrements its prompt's response_count.
func (h *Handler) DeleteResponse(w http.ResponseWriter, r *http.Request) {
	responseID, ok := pathID(w, r, "responseID", "response_id")
	if !ok {
		return
	}

	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	if err := h.Repo.DeleteResponse(r.Context(), responseID, userID); err != nil {
		handleAppError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

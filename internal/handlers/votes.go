package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/algoamigoo/micdrop/internal/middleware"
)

type setVoteRequest struct {
	Vote string `json:"vote"`
}

// SetPromptVote handles PUT /api/v1/prompts/{postID}/vote.
// Body: {"vote":"upvote"|"downvote"|"none"} — the desired end state (idempotent).
func (h *Handler) SetPromptVote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	postID, ok := pathID(w, r, "postID", "post_id")
	if !ok {
		return
	}

	var req setVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	prompt, err := h.Repo.SetPromptVote(r.Context(), userID, postID, req.Vote)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, prompt)
}

// SetResponseVote handles PUT /api/v1/responses/{responseID}/vote.
func (h *Handler) SetResponseVote(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	responseID, ok := pathID(w, r, "responseID", "response_id")
	if !ok {
		return
	}

	var req setVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.Repo.SetResponseVote(r.Context(), userID, responseID, req.Vote)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

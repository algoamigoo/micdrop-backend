package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/algoamigoo/micdrop/internal/middleware"
	"github.com/go-chi/chi/v5"
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

	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid post_id format")
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

	responseIDStr := chi.URLParam(r, "responseID")
	responseID, err := strconv.ParseInt(responseIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid response_id format")
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

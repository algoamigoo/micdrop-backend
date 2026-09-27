package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// getVoterID is a helper to extract the mock-auth user ID from the X-User-ID header.
func getVoterID(r *http.Request) string {
	return r.Header.Get("X-User-ID")
}

// UpvotePrompt handles POST /api/v1/prompts/{postID}/upvote
func (h *Handler) UpvotePrompt(w http.ResponseWriter, r *http.Request) {
	h.votePrompt(w, r, "upvote")
}

// DownvotePrompt handles POST /api/v1/prompts/{postID}/downvote
func (h *Handler) DownvotePrompt(w http.ResponseWriter, r *http.Request) {
	h.votePrompt(w, r, "downvote")
}

func (h *Handler) votePrompt(w http.ResponseWriter, r *http.Request, voteType string) {
	userID := getVoterID(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "X-User-ID header is required")
		return
	}

	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid post_id format")
		return
	}

	// The repository handles the transaction (insert vote, bump counter, update karma)
	prompt, err := h.Repo.VoteOnPrompt(r.Context(), userID, postID, voteType)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, prompt)
}

// UpvoteResponse handles POST /api/v1/responses/{responseID}/upvote
func (h *Handler) UpvoteResponse(w http.ResponseWriter, r *http.Request) {
	h.voteResponse(w, r, "upvote")
}

// DownvoteResponse handles POST /api/v1/responses/{responseID}/downvote
func (h *Handler) DownvoteResponse(w http.ResponseWriter, r *http.Request) {
	h.voteResponse(w, r, "downvote")
}

func (h *Handler) voteResponse(w http.ResponseWriter, r *http.Request, voteType string) {
	userID := getVoterID(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "X-User-ID header is required")
		return
	}

	responseIDStr := chi.URLParam(r, "responseID")
	responseID, err := strconv.ParseInt(responseIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid response_id format")
		return
	}

	resp, err := h.Repo.VoteOnResponse(r.Context(), userID, responseID, voteType)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

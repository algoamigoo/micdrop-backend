package handlers

import (
	"net/http"
	"strconv"

	"github.com/algoamigoo/micdrop/internal/middleware"
	"github.com/go-chi/chi/v5"
)


func (h *Handler) UpvotePrompt(w http.ResponseWriter, r *http.Request) {
	h.votePrompt(w, r, "upvote")
}

func (h *Handler) DownvotePrompt(w http.ResponseWriter, r *http.Request) {
	h.votePrompt(w, r, "downvote")
}

func (h *Handler) votePrompt(w http.ResponseWriter, r *http.Request, voteType string) {
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

	prompt, err := h.Repo.VoteOnPrompt(r.Context(), userID, postID, voteType)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, prompt)
}

func (h *Handler) UpvoteResponse(w http.ResponseWriter, r *http.Request) {
	h.voteResponse(w, r, "upvote")
}

func (h *Handler) DownvoteResponse(w http.ResponseWriter, r *http.Request) {
	h.voteResponse(w, r, "downvote")
}

func (h *Handler) voteResponse(w http.ResponseWriter, r *http.Request, voteType string) {
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

	resp, err := h.Repo.VoteOnResponse(r.Context(), userID, responseID, voteType)
	if err != nil {
		handleAppError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/algoamigoo/micdrop/internal/repository"
	"github.com/go-chi/chi/v5"
)

// Matches the VARCHAR(280) columns on prompts.body and responses.body, so an
// over-long body is a 400 instead of a Postgres 22001 surfaced as 500.
const maxBodyLen = 280

func validateBody(body string) string {
	if body == "" {
		return "body is required"
	}
	if n := len([]rune(body)); n > maxBodyLen {
		return fmt.Sprintf("body must be at most %d characters", maxBodyLen)
	}
	return ""
}

// pathID parses a numeric chi URL param, writing a 400 and returning ok=false on failure.
// param is the URL param name; wireName is the field name used in the error message.
func pathID(w http.ResponseWriter, r *http.Request, param, wireName string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid "+wireName+" format")
		return 0, false
	}
	return id, true
}

type postBodyRequest struct {
	Body string `json:"body"`
}

// decodePostBody reads and validates a {"body": "..."} request, writing a 400 and
// returning ok=false on failure.
func decodePostBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req postBodyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return "", false
	}
	if msg := validateBody(req.Body); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return "", false
	}
	return req.Body, true
}

// Invalid or out-of-range values are rejected with 400 rather than coerced.
func parsePagination(w http.ResponseWriter, r *http.Request, defaultLimit, maxLimit int) (limit, offset int, ok bool) {
	limit, offset = defaultLimit, 0

	if s := r.URL.Query().Get("limit"); s != "" {
		val, err := strconv.Atoi(s)
		if err != nil || val < 1 || val > maxLimit {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("limit must be an integer between 1 and %d", maxLimit))
			return 0, 0, false
		}
		limit = val
	}

	if s := r.URL.Query().Get("offset"); s != "" {
		val, err := strconv.Atoi(s)
		if err != nil || val < 0 {
			respondError(w, http.StatusBadRequest, "offset must be an integer >= 0")
			return 0, 0, false
		}
		offset = val
	}

	return limit, offset, true
}

// envelope is a standard wrapper for all our JSON responses.
type envelope map[string]any

// respondJSON sends a JSON response with a given status code.
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(envelope{"data": data}); err != nil {
			slog.Error("failed to write json response", "error", err)
		}
	}
}

// respondError sends a JSON error response.
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{"error": message})
}

// handleAppError maps repository errors to HTTP status codes. errors.Is, not ==
// because the repository wraps its errors with %w.
func handleAppError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrPromptNotFound),
		errors.Is(err, repository.ErrResponseNotFound),
		errors.Is(err, repository.ErrUserNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, repository.ErrUserIDTaken),
		errors.Is(err, repository.ErrGoogleIDTaken):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, repository.ErrInvalidVoteType),
		errors.Is(err, repository.ErrInvalidLink),
		errors.Is(err, repository.ErrSelfFollow):
		respondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrNotAuthor):
		respondError(w, http.StatusForbidden, err.Error())
	default:
		slog.Error("internal server error", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

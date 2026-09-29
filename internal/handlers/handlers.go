package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/algoamigoo/micdrop/internal/repository"
)

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

// handleAppError maps repository errors to the correct HTTP status code.
func handleAppError(w http.ResponseWriter, err error) {
	switch err {
	case repository.ErrPromptNotFound, repository.ErrResponseNotFound, repository.ErrUserNotFound:
		respondError(w, http.StatusNotFound, err.Error())
	case repository.ErrUserIDTaken, repository.ErrGoogleIDTaken:
		respondError(w, http.StatusConflict, err.Error())
	case repository.ErrInvalidVoteType, repository.ErrInvalidLink:
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		slog.Error("internal server error", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

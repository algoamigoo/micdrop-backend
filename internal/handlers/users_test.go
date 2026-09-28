package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
)

func TestGetUser_NotFound(t *testing.T) {
	repo := &InMemoryRepository{
		Err: repository.ErrUserNotFound,
	}
	router := setupTestRouter(repo)

	req := httptest.NewRequest("GET", "/api/v1/users/google_999", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestListUserPrompts_Success(t *testing.T) {
	repo := &InMemoryRepository{
		Prompts: []models.Prompt{
			{PostID: 1, UserID: "google_123", Body: "Joke 1"},
			{PostID: 2, UserID: "google_123", Body: "Joke 2"},
		},
	}
	router := setupTestRouter(repo)

	req := httptest.NewRequest("GET", "/api/v1/users/google_123/prompts", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

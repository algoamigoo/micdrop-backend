package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
)

func TestUpvoteResponse_Conflict(t *testing.T) {
	repo := &InMemoryRepository{
		Err: repository.ErrAlreadyVoted,
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("POST", "/api/v1/responses/1/upvote", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d", rr.Code)
	}
}

func TestUpvoteResponse_NotFound(t *testing.T) {
	repo := &InMemoryRepository{
		Err: repository.ErrResponseNotFound,
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("POST", "/api/v1/responses/99/upvote", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUpvoteResponse_Success(t *testing.T) {
	repo := &InMemoryRepository{
		Response: &models.Response{ResponseID: 1, UserID: "google_123", ResponseUpvotes: 1},
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("POST", "/api/v1/responses/1/upvote", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

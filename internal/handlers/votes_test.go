package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
)

func TestSetResponseVote_Success(t *testing.T) {
	up := "upvote"
	repo := &InMemoryRepository{
		Response: &models.Response{ResponseID: 1, UserID: "google_123", ResponseUpvotes: 1, ViewerVote: &up},
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("PUT", "/api/v1/responses/1/vote", strings.NewReader(`{"vote":"upvote"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if repo.SetResponseVoteCalledWith.VoteType != "upvote" {
		t.Errorf("expected vote upvote, got %q", repo.SetResponseVoteCalledWith.VoteType)
	}
}

func TestSetResponseVote_Clear(t *testing.T) {
	repo := &InMemoryRepository{
		Response: &models.Response{ResponseID: 1, ResponseUpvotes: 0},
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("PUT", "/api/v1/responses/1/vote", strings.NewReader(`{"vote":"none"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestSetResponseVote_InvalidVote(t *testing.T) {
	repo := &InMemoryRepository{
		Err: repository.ErrInvalidVoteType,
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("PUT", "/api/v1/responses/1/vote", strings.NewReader(`{"vote":"bogus"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSetResponseVote_NotFound(t *testing.T) {
	repo := &InMemoryRepository{
		Err: repository.ErrResponseNotFound,
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("PUT", "/api/v1/responses/99/vote", strings.NewReader(`{"vote":"upvote"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestSetResponseVote_Unauthorized(t *testing.T) {
	repo := &InMemoryRepository{}
	router := setupTestRouter(repo)

	req := httptest.NewRequest("PUT", "/api/v1/responses/1/vote", strings.NewReader(`{"vote":"upvote"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/algoamigoo/micdrop/internal/config"
	"github.com/algoamigoo/micdrop/internal/middleware"
	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
	"github.com/go-chi/chi/v5"
)

const jwtSecret = "test-secret"

// setupTestRouter configures a router identical to production, but with our InMemoryRepository
func setupTestRouter(repo *InMemoryRepository) http.Handler {
	h := New(repo)
	authH := NewAuthHandler(repo, &config.Config{JWTSecret: jwtSecret})
	r := chi.NewRouter()

	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Group(func(protected chi.Router) {
			protected.Use(middleware.RequireAuth(jwtSecret))
			protected.Post("/prompts", h.CreatePrompt)
			protected.Post("/prompts/{postID}/responses", h.CreateResponse)
			protected.Put("/prompts/{postID}/vote", h.SetPromptVote)
			protected.Put("/responses/{responseID}/vote", h.SetResponseVote)
			protected.Patch("/users/me", h.UpdateMe)
		})
		v1.Group(func(public chi.Router) {
			public.Use(middleware.OptionalAuth(jwtSecret))
			public.Get("/prompts", h.ListPrompts)
			public.Get("/users/{userID}", h.GetUser)
			public.Get("/users/{userID}/prompts", h.ListUserPrompts)
		})
		v1.Post("/auth/complete-signup", authH.CompleteSignup)
	})
	return r
}

func TestCreatePrompt_Unauthorized(t *testing.T) {
	repo := &InMemoryRepository{}
	router := setupTestRouter(repo)

	req := httptest.NewRequest("POST", "/api/v1/prompts", bytes.NewBufferString(`{"body":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCreatePrompt_EmptyBody(t *testing.T) {
	repo := &InMemoryRepository{}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("POST", "/api/v1/prompts", bytes.NewBufferString(`{"body":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", rr.Code)
	}
}

func TestCreatePrompt_DBError(t *testing.T) {
	repo := &InMemoryRepository{
		Err: repository.ErrUserNotFound, // Simulate user doesn't exist in DB
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("POST", "/api/v1/prompts", bytes.NewBufferString(`{"body":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing user, got %d", rr.Code)
	}
}

func TestCreatePrompt_Success(t *testing.T) {
	repo := &InMemoryRepository{
		Prompt: &models.Prompt{PostID: 1, UserID: "123", Body: "test"},
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("POST", "/api/v1/prompts", bytes.NewBufferString(`{"body":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify the handler actually pulled the user_id from the JWT context, not the body
	if repo.CreatePromptCalledWith.UserID != "google_123" {
		t.Errorf("expected repo to receive google_123, got %s", repo.CreatePromptCalledWith.UserID)
	}

	var resp map[string]models.Prompt
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["data"].Body != "test" {
		t.Errorf("unexpected response body")
	}
}

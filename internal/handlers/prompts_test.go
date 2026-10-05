package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCreatePrompt_BodyTooLong(t *testing.T) {
	repo := &InMemoryRepository{}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	body, _ := json.Marshal(map[string]string{"body": strings.Repeat("a", maxBodyLen+1)})
	req := httptest.NewRequest("POST", "/api/v1/prompts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for over-long body, got %d", rr.Code)
	}
	if repo.Prompt != nil {
		t.Error("repository should not be called for an over-long body")
	}
}

func TestCreatePrompt_BodyAtMaxLength(t *testing.T) {
	repo := &InMemoryRepository{
		Prompt: &models.Prompt{PostID: 1, UserID: "google_123", Body: strings.Repeat("a", maxBodyLen)},
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	body, _ := json.Marshal(map[string]string{"body": strings.Repeat("a", maxBodyLen)})
	req := httptest.NewRequest("POST", "/api/v1/prompts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201 at exactly maxBodyLen, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListPrompts_InvalidPagination(t *testing.T) {
	cases := []string{"?limit=abc", "?limit=0", "?limit=51", "?offset=-1", "?offset=x"}
	for _, q := range cases {
		repo := &InMemoryRepository{}
		router := setupTestRouter(repo)
		token := generateTestToken(jwtSecret, "google_123", false)

		req := httptest.NewRequest("GET", "/api/v1/prompts"+q, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", q, rr.Code)
		}
	}
}

func TestListPrompts_ValidPagination(t *testing.T) {
	for _, q := range []string{"", "?limit=50&offset=20", "?limit=1"} {
		repo := &InMemoryRepository{Prompts: []models.Prompt{}}
		router := setupTestRouter(repo)
		token := generateTestToken(jwtSecret, "google_123", false)

		req := httptest.NewRequest("GET", "/api/v1/prompts"+q, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", q, rr.Code)
		}
	}
}

// The repository's own foreign-key -> ErrUserNotFound mapping needs a live Postgres
// and is not covered here.
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

// Repository errors arrive wrapped, so handleAppError must use errors.Is.
func TestCreatePrompt_WrappedRepoErrorIsMapped(t *testing.T) {
	repo := &InMemoryRepository{
		Err: fmt.Errorf("repository.CreatePrompt insert: %w", repository.ErrUserNotFound),
	}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "google_123", false)

	req := httptest.NewRequest("POST", "/api/v1/prompts", bytes.NewBufferString(`{"body":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for wrapped ErrUserNotFound, got %d", rr.Code)
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

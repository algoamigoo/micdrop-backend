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
			protected.Patch("/prompts/{postID}", h.UpdatePrompt)
			protected.Delete("/prompts/{postID}", h.DeletePrompt)
			protected.Patch("/responses/{responseID}", h.UpdateResponse)
			protected.Delete("/responses/{responseID}", h.DeleteResponse)
			protected.Post("/prompts/{postID}/responses", h.CreateResponse)
			protected.Put("/prompts/{postID}/vote", h.SetPromptVote)
			protected.Put("/responses/{responseID}/vote", h.SetResponseVote)
			protected.Patch("/users/me", h.UpdateMe)
			protected.Put("/users/{userID}/follow", h.FollowUser)
			protected.Delete("/users/{userID}/follow", h.UnfollowUser)
		})
		v1.Group(func(public chi.Router) {
			public.Use(middleware.OptionalAuth(jwtSecret))
			public.Get("/prompts", h.ListPrompts)
			public.Get("/users/{userID}", h.GetUser)
			public.Get("/users/{userID}/prompts", h.ListUserPrompts)
			public.Get("/users/{userID}/followers", h.ListFollowers)
			public.Get("/users/{userID}/following", h.ListFollowing)
		})
		v1.Post("/auth/complete-signup", authH.CompleteSignup)
		v1.Get("/auth/google/login", authH.GoogleLogin)
		v1.Get("/auth/google/callback", authH.GoogleCallback)
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

func TestUpdatePrompt_Unauthorized(t *testing.T) {
	rr := postRequest(t, "PATCH", "/api/v1/prompts/1", "", `{"body":"edited"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestUpdatePrompt_Success(t *testing.T) {
	repo := &InMemoryRepository{Prompt: &models.Prompt{PostID: 1, Body: "edited"}}
	token := generateTestToken(jwtSecret, "alice", false)

	req := httptest.NewRequest("PATCH", "/api/v1/prompts/1", strings.NewReader(`{"body":"edited"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if repo.UpdatePromptCalledWith.PostID != 1 {
		t.Errorf("expected post_id 1, got %d", repo.UpdatePromptCalledWith.PostID)
	}
	if repo.UpdatePromptCalledWith.UserID != "alice" {
		t.Errorf("expected the author from the JWT, got %q", repo.UpdatePromptCalledWith.UserID)
	}
	if repo.UpdatePromptCalledWith.Body != "edited" {
		t.Errorf("expected body %q, got %q", "edited", repo.UpdatePromptCalledWith.Body)
	}
}

func TestUpdatePrompt_InvalidBody(t *testing.T) {
	token := generateTestToken(jwtSecret, "alice", false)
	cases := map[string]string{
		"empty":    `{"body":""}`,
		"too long": fmt.Sprintf(`{"body":%q}`, strings.Repeat("a", maxBodyLen+1)),
		"not json": `{`,
		"no field": `{}`,
	}
	for name, body := range cases {
		rr := postRequest(t, "PATCH", "/api/v1/prompts/1", token, body)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, rr.Code)
		}
	}
}

func TestUpdatePrompt_InvalidPostID(t *testing.T) {
	token := generateTestToken(jwtSecret, "alice", false)
	rr := postRequest(t, "PATCH", "/api/v1/prompts/abc", token, `{"body":"edited"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for a non-numeric post_id, got %d", rr.Code)
	}
}

func TestUpdatePrompt_NotAuthor(t *testing.T) {
	repo := &InMemoryRepository{Err: repository.ErrNotAuthor}
	token := generateTestToken(jwtSecret, "mallory", false)

	req := httptest.NewRequest("PATCH", "/api/v1/prompts/1", strings.NewReader(`{"body":"edited"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for a non-author, got %d", rr.Code)
	}
}

func TestUpdatePrompt_NotFound(t *testing.T) {
	repo := &InMemoryRepository{Err: repository.ErrPromptNotFound}
	token := generateTestToken(jwtSecret, "alice", false)

	req := httptest.NewRequest("PATCH", "/api/v1/prompts/999", strings.NewReader(`{"body":"edited"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeletePrompt_Success(t *testing.T) {
	repo := &InMemoryRepository{}
	token := generateTestToken(jwtSecret, "alice", false)

	req := httptest.NewRequest("DELETE", "/api/v1/prompts/7", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Errorf("expected an empty body, got %q", rr.Body.String())
	}
	if repo.DeletePromptCalledWith.PostID != 7 || repo.DeletePromptCalledWith.UserID != "alice" {
		t.Errorf("unexpected repo args: %+v", repo.DeletePromptCalledWith)
	}
}

func TestDeletePrompt_Unauthorized(t *testing.T) {
	rr := postRequest(t, "DELETE", "/api/v1/prompts/7", "", "")
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestDeletePrompt_NotAuthor(t *testing.T) {
	repo := &InMemoryRepository{Err: repository.ErrNotAuthor}
	token := generateTestToken(jwtSecret, "mallory", false)

	req := httptest.NewRequest("DELETE", "/api/v1/prompts/7", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestDeletePrompt_NotFound(t *testing.T) {
	repo := &InMemoryRepository{Err: repository.ErrPromptNotFound}
	token := generateTestToken(jwtSecret, "alice", false)

	req := httptest.NewRequest("DELETE", "/api/v1/prompts/404", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeletePrompt_NoEnvelope(t *testing.T) {
	repo := &InMemoryRepository{}
	token := generateTestToken(jwtSecret, "alice", false)

	req := httptest.NewRequest("DELETE", "/api/v1/prompts/7", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	setupTestRouter(repo).ServeHTTP(rr, req)

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err == nil {
		t.Errorf("expected no JSON body, got %v", payload)
	}
}

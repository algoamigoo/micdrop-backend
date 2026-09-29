package handlers

import (
	"bytes"
	"encoding/json"
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

func TestGetUser_Success(t *testing.T) {
	repo := &InMemoryRepository{
		User:  &models.User{UserID: "alice", UserName: "alice"},
		Stats: &models.UserStats{PromptCount: 3, ResponseCount: 5},
	}
	router := setupTestRouter(repo)

	req := httptest.NewRequest("GET", "/api/v1/users/alice", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data struct {
			User  models.User `json:"user"`
			Stats struct {
				PromptCount   int `json:"prompt_count"`
				ResponseCount int `json:"response_count"`
			} `json:"stats"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.User.UserID != "alice" {
		t.Errorf("expected user alice, got %q", resp.Data.User.UserID)
	}
	if resp.Data.Stats.PromptCount != 3 || resp.Data.Stats.ResponseCount != 5 {
		t.Errorf("unexpected stats: %+v", resp.Data.Stats)
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

func TestUpdateMe_Unauthorized(t *testing.T) {
	router := setupTestRouter(&InMemoryRepository{User: &models.User{UserID: "alice"}})

	req := httptest.NewRequest("PATCH", "/api/v1/users/me", bytes.NewBufferString(`{"user_name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestUpdateMe_Success(t *testing.T) {
	repo := &InMemoryRepository{User: &models.User{UserID: "alice"}}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "alice", false)

	body := `{"user_name":"Alice A.","bio":"hello","links":[{"type":"twitter","url":"https://x.com/alice"}]}`
	req := httptest.NewRequest("PATCH", "/api/v1/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if repo.UpdateProfileCalledWith.UserID != "alice" {
		t.Errorf("expected repo to receive alice, got %s", repo.UpdateProfileCalledWith.UserID)
	}
	if got := repo.UpdateProfileCalledWith.Input.UserName; got == nil || *got != "Alice A." {
		t.Errorf("unexpected user_name: %+v", got)
	}
	if got := repo.UpdateProfileCalledWith.Input.Links; got == nil || len(*got) != 1 {
		t.Errorf("unexpected links: %+v", got)
	}
}

func TestUpdateMe_InvalidLink(t *testing.T) {
	repo := &InMemoryRepository{User: &models.User{UserID: "alice"}}
	router := setupTestRouter(repo)
	token := generateTestToken(jwtSecret, "alice", false)

	body := `{"links":[{"type":"myspace","url":"https://myspace.com/alice"}]}`
	req := httptest.NewRequest("PATCH", "/api/v1/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/algoamigoo/micdrop/internal/models"
	"github.com/algoamigoo/micdrop/internal/repository"
	"github.com/golang-jwt/jwt/v5"
)

func generateOnboardingToken(secret string, googleID string) string {
	claims := jwt.MapClaims{
		"purpose":   "onboarding",
		"google_id": googleID,
		"iat":       time.Now().Unix(),
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

func signupRequest(t *testing.T, router http.Handler, bearer, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/auth/complete-signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestCompleteSignup_Success(t *testing.T) {
	repo := &InMemoryRepository{
		User: &models.User{UserID: "alice", UserName: "alice"},
	}
	router := setupTestRouter(repo)

	rr := signupRequest(t, router, generateOnboardingToken(jwtSecret, "google-123"), `{"user_id":"alice"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if repo.CreateUserCalledWith.UserID != "alice" || repo.CreateUserCalledWith.GoogleID != "google-123" {
		t.Errorf("unexpected CreateUser args: %+v", repo.CreateUserCalledWith)
	}

	var resp struct {
		Data struct {
			Token string      `json:"token"`
			User  models.User `json:"user"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Token == "" {
		t.Error("expected a session token in the response")
	}
	if resp.Data.User.UserID != "alice" {
		t.Errorf("expected user alice, got %q", resp.Data.User.UserID)
	}
}

func TestCompleteSignup_UserIDTaken(t *testing.T) {
	repo := &InMemoryRepository{
		User: &models.User{UserID: "alice"},
		Err:  repository.ErrUserIDTaken,
	}
	router := setupTestRouter(repo)

	rr := signupRequest(t, router, generateOnboardingToken(jwtSecret, "google-123"), `{"user_id":"alice"}`)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestCompleteSignup_TooShort(t *testing.T) {
	rr := signupRequest(t, setupTestRouter(&InMemoryRepository{}),
		generateOnboardingToken(jwtSecret, "google-123"), `{"user_id":"ab"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCompleteSignup_BadChars(t *testing.T) {
	rr := signupRequest(t, setupTestRouter(&InMemoryRepository{}),
		generateOnboardingToken(jwtSecret, "google-123"), `{"user_id":"hi there"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCompleteSignup_Reserved(t *testing.T) {
	rr := signupRequest(t, setupTestRouter(&InMemoryRepository{}),
		generateOnboardingToken(jwtSecret, "google-123"), `{"user_id":"admin"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCompleteSignup_MissingHeader(t *testing.T) {
	rr := signupRequest(t, setupTestRouter(&InMemoryRepository{}), "", `{"user_id":"alice"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCompleteSignup_RejectsSessionToken(t *testing.T) {
	rr := signupRequest(t, setupTestRouter(&InMemoryRepository{}),
		generateTestToken(jwtSecret, "alice", false), `{"user_id":"alice"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

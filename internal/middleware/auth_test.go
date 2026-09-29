package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

// dummyHandler reads the userID from context and writes it to the response
func dummyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(userID))
	})
}

func generateTestToken(secret string, userID string, expired bool) string {
	claims := jwt.MapClaims{
		"purpose": "session",
		"user_id": userID,
		"iat":     time.Now().Unix(),
	}
	if expired {
		claims["exp"] = time.Now().Add(-1 * time.Hour).Unix()
	} else {
		claims["exp"] = time.Now().Add(1 * time.Hour).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

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

func TestRequireAuth_MissingHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler := RequireAuth(testSecret)(dummyHandler())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuth_InvalidScheme(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	rr := httptest.NewRecorder()

	handler := RequireAuth(testSecret)(dummyHandler())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rr := httptest.NewRecorder()

	handler := RequireAuth(testSecret)(dummyHandler())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken(testSecret, "user-1", true))
	rr := httptest.NewRecorder()

	handler := RequireAuth(testSecret)(dummyHandler())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRequireAuth_RejectsOnboardingPurpose(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+generateOnboardingToken(testSecret, "google-123"))
	rr := httptest.NewRecorder()

	handler := RequireAuth(testSecret)(dummyHandler())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for onboarding-purpose token, got %d", rr.Code)
	}
}

func TestRequireAuth_ValidSessionToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken(testSecret, "user-1", false))
	rr := httptest.NewRecorder()

	handler := RequireAuth(testSecret)(dummyHandler())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "user-1" {
		t.Errorf("expected user-1 in context, got %q", rr.Body.String())
	}
}

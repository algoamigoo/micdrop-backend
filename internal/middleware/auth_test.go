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

// Local to the middleware package
func generateTestToken(secret string, userID string, expired bool) string {
	claims := jwt.MapClaims{
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
		t.Errorf("expected 401 for wrong scheme, got %d", rr.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalidtoken123")
	rr := httptest.NewRecorder()

	handler := RequireAuth(testSecret)(dummyHandler())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", rr.Code)
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	token := generateTestToken(testSecret, "user_123", true)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler := RequireAuth(testSecret)(dummyHandler())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", rr.Code)
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	token := generateTestToken(testSecret, "user_123", false)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler := RequireAuth(testSecret)(dummyHandler())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid token, got %d", rr.Code)
	}
	if rr.Body.String() != "user_123" {
		t.Errorf("expected body to be user_123, got %s", rr.Body.String())
	}
}

package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// generateTestToken creates a valid session JWT for testing authenticated routes
func generateTestToken(secret string, userID string, expired bool) string {
	claims := jwt.MapClaims{
		"purpose": "session",
		"user_id": userID,
		"iat":     time.Now().Unix(),
	}
	if expired {
		claims["exp"] = time.Now().Add(-1 * time.Hour).Unix() // Expired 1 hour ago
	} else {
		claims["exp"] = time.Now().Add(1 * time.Hour).Unix() // Valid for 1 hour
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

func postRequest(t *testing.T, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	setupTestRouter(&InMemoryRepository{}).ServeHTTP(rr, req)
	return rr
}

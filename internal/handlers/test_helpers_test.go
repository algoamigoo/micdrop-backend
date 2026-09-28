package handlers

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// generateTestToken creates a valid JWT for testing authenticated routes
func generateTestToken(secret string, userID string, expired bool) string {
	claims := jwt.MapClaims{
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

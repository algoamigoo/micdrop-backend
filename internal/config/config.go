package config

import (
	"os"
	"strings"
)

type Config struct {
	Port           string
	DatabaseURL    string
	AllowedOrigins []string
	LogLevel       string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	JWTSecret          string
	FrontendURL        string
}

func Load() *Config {
	port := getenv("PORT", "3000")
	dbURL := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/micdrop?sslmode=disable")
	origins := strings.Split(getenv("ALLOWED_ORIGINS", "*"), ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	return &Config{
		Port:           port,
		DatabaseURL:    dbURL,
		AllowedOrigins: origins,
		LogLevel:       getenv("LOG_LEVEL", "info"),

		GoogleClientID:     getenv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getenv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getenv("GOOGLE_REDIRECT_URL", "http://localhost:3000/api/v1/auth/google/callback"),
		JWTSecret:          getenv("JWT_SECRET", "supersecretjwtkey"),
		FrontendURL:        getenv("FRONTEND_URL", "http://localhost:5173/auth/callback"),
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

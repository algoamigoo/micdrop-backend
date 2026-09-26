package config

import (
	"os"
	"strings"
)

type Config struct {
	Port           string
	DatabaseURL    string
	AllowedOrigins []string
	AutoMigrate    bool
	LogLevel       string
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
		AutoMigrate:    getenv("AUTO_MIGRATE", "true") == "true",
		LogLevel:       getenv("LOG_LEVEL", "info"),
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

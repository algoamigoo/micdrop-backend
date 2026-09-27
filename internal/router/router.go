package router

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/algoamigoo/micdrop/internal/handlers"
	"github.com/algoamigoo/micdrop/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// New builds the HTTP router for the API. Domain handlers are wired
// in under /api/v1
func New(repo *repository.Repository, logger *slog.Logger, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization", "X-User-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(requestLogger(logger))

	r.Get("/healthz", healthCheck)

	// Initialize our handlers
	h := handlers.New(repo)

	r.Route("/api/v1", func(v1 chi.Router) {
		// Users
		v1.Post("/users", h.CreateUser)
		v1.Get("/users/{userID}", h.GetUser)

		// Prompts
		v1.Post("/prompts", h.CreatePrompt)
		v1.Get("/prompts", h.ListPrompts)
		v1.Get("/prompts/{postID}", h.GetPrompt)

		// Prompt Votes
		v1.Post("/prompts/{postID}/upvote", h.UpvotePrompt)
		v1.Post("/prompts/{postID}/downvote", h.DownvotePrompt)

		// Responses
		v1.Post("/prompts/{postID}/responses", h.CreateResponse)
		v1.Get("/prompts/{postID}/responses", h.ListResponses)

		// Response Votes
		v1.Post("/responses/{responseID}/upvote", h.UpvoteResponse)
		v1.Post("/responses/{responseID}/downvote", h.DownvoteResponse)
	})
	return r
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Error("encode health check", "error", err)
	}
}

// requestLogger logs one structured line per request: method, path, status,
// response size, latency
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			next.ServeHTTP(ww, req)
			logger.Info("http_request",
				"method", req.Method,
				"path", req.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(req.Context()),
			)
		})
	}
}

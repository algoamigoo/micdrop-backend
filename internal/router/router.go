package router

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/algoamigoo/micdrop/internal/config"
	"github.com/algoamigoo/micdrop/internal/handlers"
	mw "github.com/algoamigoo/micdrop/internal/middleware"
	"github.com/algoamigoo/micdrop/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func New(repo *repository.Repository, logger *slog.Logger, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(requestLogger(logger))

	r.Get("/healthz", healthCheck)

	h := handlers.New(repo)
	authH := handlers.NewAuthHandler(repo, cfg)

	r.Route("/api/v1", func(v1 chi.Router) {
		// Auth Routes (Public)
		v1.Get("/auth/google/login", authH.GoogleLogin)
		v1.Get("/auth/google/callback", authH.GoogleCallback)
		v1.Post("/auth/complete-signup", authH.CompleteSignup)

		//Authenticated routes
		v1.Group(func(protected chi.Router) {
			protected.Use(mw.RequireAuth(cfg.JWTSecret))

			protected.Get("/auth/me", authH.GetMe)
			protected.Patch("/users/me", h.UpdateMe)

			// Prompts (Create requires auth)
			protected.Post("/prompts", h.CreatePrompt)

			// Responses (Create requires auth)
			protected.Post("/prompts/{postID}/responses", h.CreateResponse)

			// Votes (Require auth, idempotent: client sends desired end state)
			protected.Put("/prompts/{postID}/vote", h.SetPromptVote)
			protected.Put("/responses/{responseID}/vote", h.SetResponseVote)
		})

		// Public Routes (viewer-aware via OptionalAuth: logged-in users get viewer_vote)
		v1.Group(func(public chi.Router) {
			public.Use(mw.OptionalAuth(cfg.JWTSecret))

			public.Get("/users/{userID}", h.GetUser)
			public.Get("/users/{userID}/prompts", h.ListUserPrompts)
			public.Get("/users/{userID}/responses", h.ListUserResponses)

			public.Get("/prompts", h.ListPrompts)
			public.Get("/prompts/{postID}", h.GetPrompt)
			public.Get("/prompts/{postID}/responses", h.ListResponses)
		})
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
// response size, latency. Logging happens in a defer so panicking requests are
// still recorded; the panic is re-raised for the Recoverer.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			defer func() {
				if r := recover(); r != nil {
					logger.Error("http_request_panic",
						"method", req.Method,
						"path", req.URL.Path,
						"duration_ms", time.Since(start).Milliseconds(),
						"request_id", middleware.GetReqID(req.Context()),
						"panic", r,
					)
					panic(r)
				}
				logger.Info("http_request",
					"method", req.Method,
					"path", req.URL.Path,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration_ms", time.Since(start).Milliseconds(),
					"request_id", middleware.GetReqID(req.Context()),
				)
			}()
			next.ServeHTTP(ww, req)
		})
	}
}

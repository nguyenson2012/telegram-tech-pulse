package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

// RouterConfig contains dependencies for the HTTP server router.
type RouterConfig struct {
	DB              *gorm.DB
	FeedHandler     *FeedHandler
	SearchHandler   *SearchHandler
	PipelineHandler *PipelineHandler
}

// NewRouter sets up chi router with middleware and all REST API routes.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Global Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Basic CORS
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Health Check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "connected"
		if cfg.DB != nil {
			sqlDB, err := cfg.DB.DB()
			if err != nil || sqlDB.Ping() != nil {
				dbStatus = "disconnected"
			}
		}

		RespondJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"database":  dbStatus,
			"timestamp": time.Now().UTC(),
		})
	})

	// API v1 routes
	r.Route("/api/v1", func(api chi.Router) {
		// Feeds
		api.Route("/feeds", func(f chi.Router) {
			f.Post("/", cfg.FeedHandler.AddFeed)
			f.Get("/", cfg.FeedHandler.ListFeeds)
			f.Patch("/{id}", cfg.FeedHandler.ToggleFeed)
		})

		// Semantic Vector Search
		api.Get("/search", cfg.SearchHandler.Search)

		// Digest Pipeline Run
		api.Post("/pipeline/run", cfg.PipelineHandler.Run)
	})

	return r
}

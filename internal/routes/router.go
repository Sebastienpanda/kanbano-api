package routes

import (
	"kanbano-api/internal/storage"
	"kanbano-api/internal/ws"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRouter(pool *pgxpool.Pool, store *storage.Client) *chi.Mux {
	allowedOrigins := getAllowedOrigins()
	ws.SetAllowedOrigins(allowedOrigins)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	RegisterRoutes(r, pool, store)
	return r
}

func getAllowedOrigins() []string {
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" {
		return []string{"http://localhost:4200"}
	}
	return strings.Split(origins, ",")
}

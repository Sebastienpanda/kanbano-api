package routes

import (
	appmiddleware "kanbano-api/internal/middleware"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/storage"
	"kanbano-api/internal/ws"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"
)

func SetupRouter(pool *pgxpool.Pool, store *storage.Client) *chi.Mux {
	allowedOrigins := getAllowedOrigins()
	ws.SetAllowedOrigins(allowedOrigins)

	logRepo := repository.NewLogRepository(pool)

	r := chi.NewRouter()
	r.Use(appmiddleware.WithRequestID)
	r.Use(appmiddleware.NewRequestLogger(logRepo))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))
	appmiddleware.SetTrustedProxies(getTrustedProxies())
	r.Use(appmiddleware.NoStore)
	r.Use(appmiddleware.RateLimit(rate.Limit(10), 30))

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

func getTrustedProxies() []string {
	raw := os.Getenv("TRUSTED_PROXY_CIDRS")
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

package main

import (
	"context"
	"kanbano-api/internal/routes"
	"log"
	"net/http"
	"os"

	appMiddleware "kanbano-api/internal/middleware"
	"kanbano-api/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {

	_ = godotenv.Load()

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to create database connection pool: %v", err)
	}
	defer pool.Close()

	err = pool.Ping(context.Background())
	if err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("database connection established")

	err = appMiddleware.InitJWKS(os.Getenv("NEON_AUTH_JWKS_URL"))
	if err != nil {
		log.Fatalf("failed to initialize JWKS: %v", err)
	}
	log.Println("JWKS initialized successfully")

	allowedOrigins := []string{"http://localhost:4200", "https://kanban.kanbano.fr"}
	ws.SetAllowedOrigins(allowedOrigins)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	routes.RegisterRoutes(r, pool)

	log.Println("server listening on port 3000")

	err = http.ListenAndServe(":3000", r)
	if err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

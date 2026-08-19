package main

import (
	"context"
	"kanbano-api/internal/routes"
	"log"
	"net/http"
	"os"

	appMiddleware "kanbano-api/internal/middleware"

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
		log.Fatal("Connexion DB échouée:", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("DB ne répond pas:", err)
	}
	log.Println("DB connectée ✅")

	err = appMiddleware.InitJWKS(os.Getenv("NEON_AUTH_JWKS_URL"))
	log.Println("JWKS initialisé ✅")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:4200", "https://kanban.kanbano.fr"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	routes.RegisterRoutes(r, pool)

	log.Println("Server running in :3000")

	err = http.ListenAndServe(":3000", r)
	if err != nil {
		return
	}
}

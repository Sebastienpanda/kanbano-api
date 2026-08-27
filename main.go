package main

import (
	"context"
	"kanbano-api/internal/routes"
	"kanbano-api/internal/storage"
	"log"
	"net/http"
	"os"
	"strconv"

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

	useSSL, _ := strconv.ParseBool(os.Getenv("GARAGE_USE_SSL"))
	store, err := storage.New(context.Background(), storage.Config{
		Endpoint:      os.Getenv("GARAGE_ENDPOINT"),
		Region:        os.Getenv("GARAGE_REGION"),
		AccessKey:     os.Getenv("GARAGE_ACCESS_KEY"),
		SecretKey:     os.Getenv("GARAGE_SECRET_KEY"),
		Bucket:        os.Getenv("GARAGE_BUCKET"),
		PublicBaseURL: os.Getenv("GARAGE_PUBLIC_BASE_URL"),
		UseSSL:        useSSL,
	})
	if err != nil {
		log.Printf("object storage unavailable, avatar endpoints disabled: %v", err)
		store = nil
	} else {
		log.Println("object storage connected")
	}

	allowedOrigins := []string{"http://localhost:4200", "https://kanban.kanbano.fr"}
	ws.SetAllowedOrigins(allowedOrigins)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	routes.RegisterRoutes(r, pool, store)

	log.Println("server listening on port 3000")

	err = http.ListenAndServe(":3000", r)
	if err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

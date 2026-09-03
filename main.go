package main

import (
	"kanbano-api/internal/db"
	middleware2 "kanbano-api/internal/middleware"
	"kanbano-api/internal/routes"
	"kanbano-api/internal/server"
	"kanbano-api/internal/storage"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	_ = godotenv.Load()

	pool := db.MustConnectDB()
	defer pool.Close()

	middleware2.MustInitJWKS(os.Getenv("NEON_AUTH_JWKS_URL"))
	store := storage.ConnectStorage()

	r := routes.SetupRouter(pool, store)
	server.Run(r)
}

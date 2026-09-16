package main

import (
	"kanbano-api/internal/db"
	middleware2 "kanbano-api/internal/middleware"
	"kanbano-api/internal/routes"
	"kanbano-api/internal/server"
	"kanbano-api/internal/storage"
	"os"
	"strconv"

	_ "image/jpeg" // register JPEG decoder used by internal/media for avatar processing

	"github.com/joho/godotenv"
)

// @title Kanbano API
// @version 1.0
// @description REST API for the Kanbano kanban application (workspaces, columns, tasks, tags, organisations, users).
// @host localhost:3000
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Neon Auth JWT (EdDSA), sent as "Bearer <token>".
func main() {
	_ = godotenv.Load()

	pool := db.MustConnectDB()
	defer pool.Close()

	middleware2.MustInitJWKS(os.Getenv("NEON_AUTH_JWKS_URL"))
	middleware2.InitAdminIDs(os.Getenv("ADMIN_USER_IDS"))
	middleware2.InitSlowRequestThreshold(slowRequestThresholdMs())
	store := storage.ConnectStorage()

	r := routes.SetupRouter(pool, store)
	server.Run(r)
}

func slowRequestThresholdMs() int {
	raw := os.Getenv("SLOW_REQUEST_THRESHOLD_MS")
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return 1000
	}
	return ms
}

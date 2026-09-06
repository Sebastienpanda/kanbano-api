package main

import (
	"kanbano-api/internal/db"
	middleware2 "kanbano-api/internal/middleware"
	"kanbano-api/internal/routes"
	"kanbano-api/internal/server"
	"kanbano-api/internal/storage"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

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

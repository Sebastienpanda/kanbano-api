package middleware

import (
	"kanbano-api/internal/logging"
	"log/slog"
	"os"
)

func MustInitJWKS(jwksURL string) {
	if err := InitJWKS(jwksURL); err != nil {
		logging.Logger.Error("failed to initialize JWKS", slog.Any("error", err))
		os.Exit(1)
	}
	logging.Logger.Info("JWKS initialized successfully")
}

package db

import (
	"context"
	"kanbano-api/internal/logging"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func MustConnectDB() *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		logging.Logger.Error("failed to create database connection pool", slog.Any("error", err))
		os.Exit(1)
	}
	if err := pool.Ping(context.Background()); err != nil {
		logging.Logger.Error("database ping failed", slog.Any("error", err))
		os.Exit(1)
	}
	logging.Logger.Info("database connection established")
	return pool
}

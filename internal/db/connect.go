package db

import (
	"context"
	"kanbano-api/internal/logging"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func MustConnectDB() *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		logging.Logger.Error("failed to parse database connection string", slog.Any("error", err))
		os.Exit(1)
	}
	// Neon's pooled endpoint runs PgBouncer in transaction mode, which does not
	// support server-side prepared statements shared across pooled connections.
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
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

package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LogRepository struct {
	db *pgxpool.Pool
}

func NewLogRepository(db *pgxpool.Pool) *LogRepository {
	return &LogRepository{db: db}
}

func (r *LogRepository) List(ctx context.Context, level *string, limit, offset int) ([]models.Log, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			level,
			message,
			source,
			user_id,
			request_id,
			metadata,
			created_at
		FROM logs
		WHERE ($1::text IS NULL OR level = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
		`,
		level,
		limit,
		offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Log])
}

func (r *LogRepository) Insert(ctx context.Context, level, message, source string, userID, requestID *uuid.UUID, metadata []byte) error {
	// With QueryExecModeSimpleProtocol, pgx encodes []byte as a bytea literal
	// ('\x7b...'), which PostgreSQL rejects for a json column: send it as text.
	var metadataText *string
	if metadata != nil {
		s := string(metadata)
		metadataText = &s
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO logs (level, message, source, user_id, request_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		`,
		level,
		message,
		source,
		userID,
		requestID,
		metadataText)
	return err
}

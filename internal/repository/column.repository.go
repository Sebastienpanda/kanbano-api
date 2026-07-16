package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ColumnRepository struct {
	db *pgxpool.Pool
}

func NewColumnRepository(db *pgxpool.Pool) *ColumnRepository {
	return &ColumnRepository{db: db}
}

func (r *ColumnRepository) Create(ctx context.Context, title string, workspaceID uuid.UUID) (models.Column, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO columns (title, workspace_id, position)
		VALUES ($1, $2, (SELECT COALESCE(MAX(position) + 1, 0) FROM columns WHERE workspace_id = $2))
		RETURNING *
	`, title, workspaceID)
	if err != nil {
		return models.Column{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Column])
}

func (r *ColumnRepository) Update(ctx context.Context, id uuid.UUID, workspaceID uuid.UUID, title *string, position *int) (models.Column, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE columns
		SET title    = COALESCE($1, title),
		    position = COALESCE($2, position),
		    updated_at = NOW()
		WHERE id = $3 AND workspace_id = $4
		RETURNING *
	`, title, position, id, workspaceID)
	if err != nil {
		return models.Column{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Column])
}

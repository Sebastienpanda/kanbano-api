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

func (r *ColumnRepository) Create(ctx context.Context, name string, workspaceID uuid.UUID) (models.Column, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO columns (name, workspace_id, position)
		VALUES ($1, $2, (SELECT COALESCE(MAX(position) + 1, 0) FROM columns WHERE workspace_id = $2))
		RETURNING *
	`, name, workspaceID)
	if err != nil {
		return models.Column{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Column])
}

func (r *ColumnRepository) ListNames(ctx context.Context, workspaceID uuid.UUID) ([]models.ColumnName, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, workspace_id, name FROM columns WHERE workspace_id = $1 AND deleted_at IS NULL ORDER BY position ASC",
		workspaceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.ColumnName])
}

func (r *ColumnRepository) Exists(ctx context.Context, id uuid.UUID, workspaceID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM columns WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL)",
		id, workspaceID).Scan(&exists)
	return exists, err
}

func (r *ColumnRepository) Update(ctx context.Context, id uuid.UUID, workspaceID uuid.UUID, name *string) (models.Column, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE columns
		SET name       = COALESCE($1, name),
		    updated_at = NOW()
		WHERE id = $2 AND workspace_id = $3 AND deleted_at IS NULL
		RETURNING *
	`, name, id, workspaceID)
	if err != nil {
		return models.Column{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Column])
}

// Reorder déplace une colonne à la position donnée et décale automatiquement
// les autres colonnes du workspace pour combler l'écart et faire de la place.
func (r *ColumnRepository) Reorder(ctx context.Context, id uuid.UUID, workspaceID uuid.UUID, position int) (models.Column, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Column{}, err
	}
	defer tx.Rollback(ctx)

	var oldPosition int
	if err := tx.QueryRow(ctx,
		"SELECT position FROM columns WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL FOR UPDATE",
		id, workspaceID).Scan(&oldPosition); err != nil {
		return models.Column{}, err
	}

	if position != oldPosition {
		if position < oldPosition {
			if _, err := tx.Exec(ctx,
				"UPDATE columns SET position = position + 1, updated_at = NOW() WHERE workspace_id = $1 AND id != $2 AND position >= $3 AND position < $4 AND deleted_at IS NULL",
				workspaceID, id, position, oldPosition); err != nil {
				return models.Column{}, err
			}
		} else {
			if _, err := tx.Exec(ctx,
				"UPDATE columns SET position = position - 1, updated_at = NOW() WHERE workspace_id = $1 AND id != $2 AND position > $3 AND position <= $4 AND deleted_at IS NULL",
				workspaceID, id, oldPosition, position); err != nil {
				return models.Column{}, err
			}
		}
	}

	rows, err := tx.Query(ctx, `
		UPDATE columns
		SET position   = $1,
		    updated_at = NOW()
		WHERE id = $2 AND workspace_id = $3 AND deleted_at IS NULL
		RETURNING *
	`, position, id, workspaceID)
	if err != nil {
		return models.Column{}, err
	}
	column, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Column])
	if err != nil {
		return models.Column{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Column{}, err
	}

	return column, nil
}

func (r *ColumnRepository) SoftDelete(ctx context.Context, id uuid.UUID, workspaceID uuid.UUID) (models.Column, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Column{}, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		UPDATE columns
		SET deleted_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		RETURNING *
	`, id, workspaceID)
	if err != nil {
		return models.Column{}, err
	}
	column, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Column])
	if err != nil {
		return models.Column{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE tasks SET deleted_at = NOW()
		WHERE column_id = $1 AND deleted_at IS NULL
	`, id); err != nil {
		return models.Column{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Column{}, err
	}

	return column, nil
}

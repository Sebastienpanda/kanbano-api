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

func (r *ColumnRepository) Create(ctx context.Context, name string, workspaceID uuid.UUID, createdBy uuid.UUID) (models.Column, error) {
	return queryStruct[models.Column](ctx, r.db, `
		INSERT INTO columns (name, workspace_id, position, created_by)
		VALUES ($1, $2, (SELECT COALESCE(MAX(position) + 1, 0) FROM columns WHERE workspace_id = $2), $3)
		RETURNING id, name, position, workspace_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		name,
		workspaceID,
		createdBy)
}

func (r *ColumnRepository) ListNames(ctx context.Context, workspaceID uuid.UUID) ([]models.ColumnName, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, workspace_id, name
		FROM columns
		WHERE workspace_id = $1 
		  AND deleted_at IS NULL
		`,
		workspaceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.ColumnName])
}

func (r *ColumnRepository) Exists(ctx context.Context, id uuid.UUID, workspaceID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM columns
			WHERE id = $1 
			  AND workspace_id = $2 
			  AND deleted_at IS NULL
		)
		`,
		id,
		workspaceID)
	err := row.Scan(&exists)
	return exists, err
}

func (r *ColumnRepository) Update(ctx context.Context, id uuid.UUID, workspaceID uuid.UUID, name *string, actorID uuid.UUID) (models.Column, error) {
	return queryStruct[models.Column](ctx, r.db, `
		UPDATE columns
		SET name       = COALESCE($1, name),
		    updated_by = $4,
		    updated_at = NOW()
		WHERE id = $2
		  AND workspace_id = $3
		  AND deleted_at IS NULL
		RETURNING id, name, position, workspace_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		name,
		id,
		workspaceID,
		actorID)
}

func (r *ColumnRepository) Reorder(ctx context.Context, id uuid.UUID, workspaceID uuid.UUID, position int, actorID uuid.UUID) (models.Column, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Column{}, err
	}
	defer tx.Rollback(ctx)

	var oldPosition int
	row := tx.QueryRow(ctx, `
		SELECT position
		FROM columns
		WHERE id = $1 
		  AND workspace_id = $2 
		  AND deleted_at IS NULL
		FOR UPDATE
		`,
		id,
		workspaceID)
	err = row.Scan(&oldPosition)
	if err != nil {
		return models.Column{}, err
	}

	if position != oldPosition {
		if position < oldPosition {
			_, err = tx.Exec(ctx, `
				UPDATE columns
				SET position = position + 1, updated_by = $5, updated_at = NOW()
				WHERE workspace_id = $1
				  AND id != $2
				  AND position >= $3
				  AND position < $4
				  AND deleted_at IS NULL
				`,
				workspaceID,
				id,
				position,
				oldPosition,
				actorID)
			if err != nil {
				return models.Column{}, err
			}
		} else {
			_, err = tx.Exec(ctx, `
				UPDATE columns
				SET position = position - 1, updated_by = $5, updated_at = NOW()
				WHERE workspace_id = $1 AND id != $2 AND position > $3 AND position <= $4 AND deleted_at IS NULL
				`,
				workspaceID,
				id,
				oldPosition,
				position,
				actorID)
			if err != nil {
				return models.Column{}, err
			}
		}
	}

	column, err := queryStruct[models.Column](ctx, tx, `
		UPDATE columns
		SET position   = $1,
		    updated_by = $4,
		    updated_at = NOW()
		WHERE id = $2
		  AND workspace_id = $3
		  AND deleted_at IS NULL
		RETURNING id, name, position, workspace_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		position,
		id,
		workspaceID,
		actorID)
	if err != nil {
		return models.Column{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return models.Column{}, err
	}

	return column, nil
}

func (r *ColumnRepository) SoftDelete(ctx context.Context, id uuid.UUID, workspaceID uuid.UUID, actorID uuid.UUID) (models.Column, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Column{}, err
	}
	defer tx.Rollback(ctx)

	column, err := queryStruct[models.Column](ctx, tx, `
		UPDATE columns
		SET deleted_at = NOW(),
		    deleted_by = $3
		WHERE id = $1
		  AND workspace_id = $2
		  AND deleted_at IS NULL
		RETURNING id, name, position, workspace_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		id,
		workspaceID,
		actorID)
	if err != nil {
		return models.Column{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE tasks
		SET deleted_at = NOW(),
		    deleted_by = $2
		WHERE column_id = $1
		  AND deleted_at IS NULL
		`,
		id,
		actorID)
	if err != nil {
		return models.Column{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return models.Column{}, err
	}

	return column, nil
}

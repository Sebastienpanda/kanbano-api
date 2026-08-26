package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, name string, description *string, columnID uuid.UUID, stateID *uuid.UUID) (models.Task, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO tasks (name, description, column_id, state_id, position)
		VALUES ($1, $2, $3, $4, (SELECT COALESCE(MAX(position) + 1, 0) FROM tasks WHERE column_id = $3))
		RETURNING id, name, description, position, column_id, state_id, created_at, updated_at, deleted_at
		`,
		name,
		description,
		columnID,
		stateID)
	if err != nil {
		return models.Task{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Task])
}

func (r *TaskRepository) Update(ctx context.Context, id uuid.UUID, columnID uuid.UUID, name *string, description *string, stateID *uuid.UUID) (models.Task, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE tasks
		SET name        = COALESCE($1, name),
		    description = COALESCE($2, description),
		    state_id    = COALESCE($3, state_id),
		    updated_at  = NOW()
		WHERE id = $4 AND column_id = $5 AND deleted_at IS NULL
		RETURNING id, name, description, position, column_id, state_id, created_at, updated_at, deleted_at
		`,
		name,
		description,
		stateID,
		id,
		columnID)
	if err != nil {
		return models.Task{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Task])
}

func (r *TaskRepository) Reorder(ctx context.Context, id uuid.UUID, columnID uuid.UUID, position *int, newColumnID *uuid.UUID) (models.Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Task{}, err
	}
	defer tx.Rollback(ctx)

	var oldPosition int
	row := tx.QueryRow(ctx, `
		SELECT position
		FROM tasks
		WHERE id = $1 AND column_id = $2 AND deleted_at IS NULL
		FOR UPDATE
		`,
		id,
		columnID)
	err = row.Scan(&oldPosition)
	if err != nil {
		return models.Task{}, err
	}

	targetColumnID := columnID
	if newColumnID != nil {
		targetColumnID = *newColumnID
	}

	var newPosition int
	if targetColumnID != columnID {
		_, err = tx.Exec(ctx, `
			UPDATE tasks
			SET position = position - 1, updated_at = NOW()
			WHERE column_id = $1 
			  AND position > $2 
			  AND deleted_at IS NULL
			`,
			columnID,
			oldPosition)
		if err != nil {
			return models.Task{}, err
		}

		if position != nil {
			newPosition = *position
		} else {
			row := tx.QueryRow(ctx, `
				SELECT COALESCE(MAX(position) + 1, 0)
				FROM tasks
				WHERE column_id = $1 AND deleted_at IS NULL
				`,
				targetColumnID)
			err = row.Scan(&newPosition)
			if err != nil {
				return models.Task{}, err
			}
		}

		_, err = tx.Exec(ctx, `
			UPDATE tasks
			SET position = position + 1, updated_at = NOW()
			WHERE column_id = $1 
			  AND position >= $2 
			  AND deleted_at IS NULL
			`,
			targetColumnID,
			newPosition)
		if err != nil {
			return models.Task{}, err
		}
	} else {
		newPosition = oldPosition
		if position != nil {
			newPosition = *position
		}
		if newPosition != oldPosition {
			if newPosition < oldPosition {
				_, err = tx.Exec(ctx, `
					UPDATE tasks
					SET position = position + 1, updated_at = NOW()
					WHERE column_id = $1 
					  AND id != $2 
					  AND position >= $3 
					  AND position < $4 
					  AND deleted_at IS NULL
					`,
					columnID,
					id,
					newPosition,
					oldPosition)
				if err != nil {
					return models.Task{}, err
				}
			} else {
				_, err = tx.Exec(ctx, `
					UPDATE tasks
					SET position = position - 1, updated_at = NOW()
					WHERE column_id = $1 
					  AND id != $2 
					  AND position > $3 
					  AND position <= $4 
					  AND deleted_at IS NULL
					`,
					columnID,
					id,
					oldPosition,
					newPosition)
				if err != nil {
					return models.Task{}, err
				}
			}
		}
	}

	rows, err := tx.Query(ctx, `
		UPDATE tasks
		SET position   = $1,
		    column_id  = $2,
		    updated_at = NOW()
		WHERE id = $3
		  AND deleted_at IS NULL
		RETURNING id, name, description, position, column_id, state_id, created_at, updated_at, deleted_at
		`,
		newPosition,
		targetColumnID,
		id)
	if err != nil {
		return models.Task{}, err
	}
	task, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Task])
	if err != nil {
		return models.Task{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return models.Task{}, err
	}

	return task, nil
}

func (r *TaskRepository) SoftDelete(ctx context.Context, id uuid.UUID, columnID uuid.UUID) (models.Task, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE tasks
		SET deleted_at = NOW()
		WHERE id = $1
		  AND column_id = $2
		  AND deleted_at IS NULL
		RETURNING id, name, description, position, column_id, state_id, created_at, updated_at, deleted_at
		`,
		id,
		columnID)
	if err != nil {
		return models.Task{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Task])
}

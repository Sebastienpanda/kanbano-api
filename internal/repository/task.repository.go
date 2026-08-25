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
		RETURNING *
	`, name, description, columnID, stateID)
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
		RETURNING *
	`, name, description, stateID, id, columnID)
	if err != nil {
		return models.Task{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Task])
}

// Reorder déplace une tâche à la position donnée, éventuellement vers une autre colonne
// (newColumnID), et décale automatiquement les autres tâches des colonnes concernées
// pour combler l'écart et faire de la place, comme un vrai déplacement dans une liste.
func (r *TaskRepository) Reorder(ctx context.Context, id uuid.UUID, columnID uuid.UUID, position *int, newColumnID *uuid.UUID) (models.Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Task{}, err
	}
	defer tx.Rollback(ctx)

	var oldPosition int
	if err := tx.QueryRow(ctx,
		"SELECT position FROM tasks WHERE id = $1 AND column_id = $2 AND deleted_at IS NULL FOR UPDATE",
		id, columnID).Scan(&oldPosition); err != nil {
		return models.Task{}, err
	}

	targetColumnID := columnID
	if newColumnID != nil {
		targetColumnID = *newColumnID
	}

	var newPosition int
	if targetColumnID != columnID {
		if _, err := tx.Exec(ctx,
			"UPDATE tasks SET position = position - 1, updated_at = NOW() WHERE column_id = $1 AND position > $2 AND deleted_at IS NULL",
			columnID, oldPosition); err != nil {
			return models.Task{}, err
		}

		if position != nil {
			newPosition = *position
		} else if err := tx.QueryRow(ctx,
			"SELECT COALESCE(MAX(position) + 1, 0) FROM tasks WHERE column_id = $1 AND deleted_at IS NULL",
			targetColumnID).Scan(&newPosition); err != nil {
			return models.Task{}, err
		}

		if _, err := tx.Exec(ctx,
			"UPDATE tasks SET position = position + 1, updated_at = NOW() WHERE column_id = $1 AND position >= $2 AND deleted_at IS NULL",
			targetColumnID, newPosition); err != nil {
			return models.Task{}, err
		}
	} else {
		newPosition = oldPosition
		if position != nil {
			newPosition = *position
		}
		if newPosition != oldPosition {
			if newPosition < oldPosition {
				if _, err := tx.Exec(ctx,
					"UPDATE tasks SET position = position + 1, updated_at = NOW() WHERE column_id = $1 AND id != $2 AND position >= $3 AND position < $4 AND deleted_at IS NULL",
					columnID, id, newPosition, oldPosition); err != nil {
					return models.Task{}, err
				}
			} else {
				if _, err := tx.Exec(ctx,
					"UPDATE tasks SET position = position - 1, updated_at = NOW() WHERE column_id = $1 AND id != $2 AND position > $3 AND position <= $4 AND deleted_at IS NULL",
					columnID, id, oldPosition, newPosition); err != nil {
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
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING *
	`, newPosition, targetColumnID, id)
	if err != nil {
		return models.Task{}, err
	}
	task, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Task])
	if err != nil {
		return models.Task{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Task{}, err
	}

	return task, nil
}

func (r *TaskRepository) SoftDelete(ctx context.Context, id uuid.UUID, columnID uuid.UUID) (models.Task, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE tasks
		SET deleted_at = NOW()
		WHERE id = $1 AND column_id = $2 AND deleted_at IS NULL
		RETURNING *
	`, id, columnID)
	if err != nil {
		return models.Task{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Task])
}

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

func (r *TaskRepository) Exists(ctx context.Context, taskID, columnID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM tasks
			WHERE id = $1 AND column_id = $2 AND deleted_at IS NULL
		)
		`,
		taskID,
		columnID)
	err := row.Scan(&exists)
	return exists, err
}

func (r *TaskRepository) Create(ctx context.Context, name string, description *string, columnID uuid.UUID, tagID *uuid.UUID, status *string, createdBy uuid.UUID) (models.Task, error) {
	return queryStruct[models.Task](ctx, r.db, `
		INSERT INTO tasks (name, description, column_id, tag_id, status, position, created_by)
		VALUES ($1, $2, $3, $4, COALESCE($5, 'À faire'), (SELECT COALESCE(MAX(position) + 1, 0) FROM tasks WHERE column_id = $3), $6)
		RETURNING id, name, description, position, column_id, tag_id, status, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		name,
		description,
		columnID,
		tagID,
		status,
		createdBy)
}

type TaskUpdate struct {
	ID          uuid.UUID
	ColumnID    uuid.UUID
	Name        *string
	Description *string
	TagID       *uuid.UUID
	Status      *string
	ActorID     uuid.UUID
}

func (r *TaskRepository) Update(ctx context.Context, update TaskUpdate) (models.Task, error) {
	return queryStruct[models.Task](ctx, r.db, `
		UPDATE tasks
		SET name        = COALESCE($1, name),
		    description = COALESCE($2, description),
		    tag_id      = COALESCE($3, tag_id),
		    status      = COALESCE($4, status),
		    updated_by  = $7,
		    updated_at  = NOW()
		WHERE id = $5 AND column_id = $6 AND deleted_at IS NULL
		RETURNING id, name, description, position, column_id, tag_id, status, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		update.Name,
		update.Description,
		update.TagID,
		update.Status,
		update.ID,
		update.ColumnID,
		update.ActorID)
}

func shiftPositions(ctx context.Context, tx pgx.Tx, columnID uuid.UUID, delta int, where string, args []any, actorID uuid.UUID) error {
	query := `
		UPDATE tasks
		SET position = position + $1, updated_by = $2, updated_at = NOW()
		WHERE column_id = $3
		  AND deleted_at IS NULL
		  AND ` + where
	fullArgs := append([]any{delta, actorID, columnID}, args...)
	_, err := tx.Exec(ctx, query, fullArgs...)
	return err
}

func reorderAcrossColumns(ctx context.Context, tx pgx.Tx, oldColumnID, targetColumnID uuid.UUID, oldPosition int, position *int, actorID uuid.UUID) (int, error) {
	if err := shiftPositions(ctx, tx, oldColumnID, -1, "position > $4", []any{oldPosition}, actorID); err != nil {
		return 0, err
	}

	newPosition := 0
	if position != nil {
		newPosition = *position
	} else {
		row := tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(position) + 1, 0)
			FROM tasks
			WHERE column_id = $1 AND deleted_at IS NULL
			`,
			targetColumnID)
		if err := row.Scan(&newPosition); err != nil {
			return 0, err
		}
	}

	if err := shiftPositions(ctx, tx, targetColumnID, 1, "position >= $4", []any{newPosition}, actorID); err != nil {
		return 0, err
	}

	return newPosition, nil
}

func reorderWithinColumn(ctx context.Context, tx pgx.Tx, id, columnID uuid.UUID, oldPosition, newPosition int, actorID uuid.UUID) error {
	if newPosition == oldPosition {
		return nil
	}

	if newPosition < oldPosition {
		return shiftPositions(ctx, tx, columnID, 1, "id != $4 AND position >= $5 AND position < $6", []any{id, newPosition, oldPosition}, actorID)
	}
	return shiftPositions(ctx, tx, columnID, -1, "id != $4 AND position > $5 AND position <= $6", []any{id, oldPosition, newPosition}, actorID)
}

func (r *TaskRepository) Reorder(ctx context.Context, id, columnID uuid.UUID, position *int, newColumnID *uuid.UUID, actorID uuid.UUID) (models.Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Task{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var oldPosition int
	row := tx.QueryRow(ctx, `
		SELECT position
		FROM tasks
		WHERE id = $1 AND column_id = $2 AND deleted_at IS NULL
		FOR UPDATE
		`,
		id,
		columnID)
	if err := row.Scan(&oldPosition); err != nil {
		return models.Task{}, err
	}

	targetColumnID := columnID
	if newColumnID != nil {
		targetColumnID = *newColumnID
	}

	var newPosition int
	if targetColumnID != columnID {
		newPosition, err = reorderAcrossColumns(ctx, tx, columnID, targetColumnID, oldPosition, position, actorID)
		if err != nil {
			return models.Task{}, err
		}
	} else {
		newPosition = oldPosition
		if position != nil {
			newPosition = *position
		}
		if err := reorderWithinColumn(ctx, tx, id, columnID, oldPosition, newPosition, actorID); err != nil {
			return models.Task{}, err
		}
	}

	task, err := queryStruct[models.Task](ctx, tx, `
		UPDATE tasks
		SET position   = $1,
		    column_id  = $2,
		    updated_by = $4,
		    updated_at = NOW()
		WHERE id = $3
		  AND deleted_at IS NULL
		RETURNING id, name, description, position, column_id, tag_id, status, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		newPosition,
		targetColumnID,
		id,
		actorID)
	if err != nil {
		return models.Task{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Task{}, err
	}

	return task, nil
}

func (r *TaskRepository) SoftDelete(ctx context.Context, id, columnID, actorID uuid.UUID) (models.Task, error) {
	return queryStruct[models.Task](ctx, r.db, `
		UPDATE tasks
		SET deleted_at = NOW(),
		    deleted_by = $3
		WHERE id = $1
		  AND column_id = $2
		  AND deleted_at IS NULL
		RETURNING id, name, description, position, column_id, tag_id, status, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		id,
		columnID,
		actorID)
}

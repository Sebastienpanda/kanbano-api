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

func (r *TaskRepository) Create(ctx context.Context, title string, description *string, columnID uuid.UUID) (models.Task, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO tasks (title, description, column_id, position)
		VALUES ($1, $2, $3, (SELECT COALESCE(MAX(position) + 1, 0) FROM tasks WHERE column_id = $3))
		RETURNING *
	`, title, description, columnID)
	if err != nil {
		return models.Task{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Task])
}

func (r *TaskRepository) Update(ctx context.Context, id uuid.UUID, columnID uuid.UUID, title *string, description *string, position *int) (models.Task, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE tasks
		SET title       = COALESCE($1, title),
		    description = COALESCE($2, description),
		    position    = COALESCE($3, position),
		    updated_at  = NOW()
		WHERE id = $4 AND column_id = $5
		RETURNING *
	`, title, description, position, id, columnID)
	if err != nil {
		return models.Task{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Task])
}

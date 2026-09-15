package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskAssigneeRepository struct {
	db *pgxpool.Pool
}

func NewTaskAssigneeRepository(db *pgxpool.Pool) *TaskAssigneeRepository {
	return &TaskAssigneeRepository{db: db}
}

func (r *TaskAssigneeRepository) Assign(ctx context.Context, taskID, memberID, assignedBy uuid.UUID, role string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO task_assignees (task_id, member_id, role, assigned_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id, member_id) DO UPDATE SET role = EXCLUDED.role
		`,
		taskID,
		memberID,
		role,
		assignedBy)
	return err
}

func (r *TaskAssigneeRepository) ListForTask(ctx context.Context, taskID uuid.UUID) ([]models.TaskAssignee, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.name, u.email, u.avatar_version, ta.role
		FROM task_assignees ta
		JOIN users u ON u.id = ta.member_id
		WHERE ta.task_id = $1
		ORDER BY ta.created_at
		`,
		taskID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.TaskAssignee])
}

func (r *TaskAssigneeRepository) HasAccess(ctx context.Context, taskID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM task_assignees
			WHERE task_id = $1 AND member_id = $2
		)
		`,
		taskID,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

func (r *TaskAssigneeRepository) Unassign(ctx context.Context, taskID, memberID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM task_assignees
		WHERE task_id = $1 AND member_id = $2
		`,
		taskID,
		memberID)
	return err
}

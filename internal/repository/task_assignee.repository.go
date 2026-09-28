package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TaskAssigneeRepository manages who works on a task. An assignment only
// shows it on the task: it grants no access and changes no role.
type TaskAssigneeRepository struct {
	db *pgxpool.Pool
}

func NewTaskAssigneeRepository(db *pgxpool.Pool) *TaskAssigneeRepository {
	return &TaskAssigneeRepository{db: db}
}

func (r *TaskAssigneeRepository) Assign(ctx context.Context, taskID, memberID, assignedBy uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO task_assignees (task_id, member_id, assigned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (task_id, member_id) DO NOTHING
		`,
		taskID,
		memberID,
		assignedBy)
	return err
}

// ListForTask returns the members assigned to the task, followed by the
// guests who accepted an invitation on it (is_guest).
func (r *TaskAssigneeRepository) ListForTask(ctx context.Context, taskID uuid.UUID) ([]models.TaskAssignee, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.name, u.email, u.avatar_version, a.is_guest
		FROM (
			SELECT member_id AS user_id, false AS is_guest, created_at
			FROM task_assignees
			WHERE task_id = $1
			UNION ALL
			SELECT user_id, true, NULL
			FROM task_guest_access
			WHERE task_id = $1
		) a
		JOIN users u ON u.id = a.user_id
		ORDER BY a.is_guest, a.created_at, u.email
		`,
		taskID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.TaskAssignee])
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

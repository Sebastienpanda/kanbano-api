package repository

import (
	"context"
	"errors"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTaskGuestInvitationAlreadyPending = errors.New("task_guest: invitation already pending for this email")

// TaskGuestRepository manages external, non-organisation-member invitations
// scoped to a single task. It is deliberately separate from RoleRepository's
// task_assignees handling: task_assignees resolves roles for existing
// organisation members, task_guests onboards people who have no
// organisation membership at all.
type TaskGuestRepository struct {
	db *pgxpool.Pool
}

func NewTaskGuestRepository(db *pgxpool.Pool) *TaskGuestRepository {
	return &TaskGuestRepository{db: db}
}

type TaskGuestInvitationParams struct {
	TaskID    uuid.UUID
	Email     string
	InvitedBy uuid.UUID
	Role      *string
}

func (r *TaskGuestRepository) CreateInvitation(ctx context.Context, params TaskGuestInvitationParams) (models.TaskGuest, error) {
	role := "view"
	if params.Role != nil {
		role = *params.Role
	}

	guest, err := queryStruct[models.TaskGuest](ctx, r.db, `
		INSERT INTO task_guests (task_id, email, invited_by, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, task_id, email, user_id, role, status, invited_by, created_at, responded_at
		`,
		params.TaskID,
		params.Email,
		params.InvitedBy,
		role)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return models.TaskGuest{}, ErrTaskGuestInvitationAlreadyPending
	}
	return guest, err
}

func (r *TaskGuestRepository) ListSentInvitations(ctx context.Context, taskID uuid.UUID, limit, offset int) ([]models.TaskGuest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, task_id, email, user_id, role, status, invited_by, created_at, responded_at
		FROM task_guests
		WHERE task_id = $1
		  AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
		`,
		taskID,
		limit,
		offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.TaskGuest])
}

func (r *TaskGuestRepository) ListReceivedInvitations(ctx context.Context, email string, limit, offset int) ([]models.TaskGuest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, task_id, email, user_id, role, status, invited_by, created_at, responded_at
		FROM task_guests
		WHERE LOWER(email) = LOWER($1)
		  AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
		`,
		email,
		limit,
		offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.TaskGuest])
}

func (r *TaskGuestRepository) AcceptInvitation(ctx context.Context, invitationID, userID uuid.UUID, email string) (models.TaskGuest, error) {
	return queryStruct[models.TaskGuest](ctx, r.db, `
		UPDATE task_guests
		SET status = 'accepted',
		    user_id = $3,
		    responded_at = NOW()
		WHERE id = $1
		  AND LOWER(email) = LOWER($2)
		  AND status = 'pending'
		RETURNING id, task_id, email, user_id, role, status, invited_by, created_at, responded_at
		`,
		invitationID,
		email,
		userID)
}

func (r *TaskGuestRepository) DeclineInvitation(ctx context.Context, invitationID uuid.UUID, email string) (models.TaskGuest, error) {
	return queryStruct[models.TaskGuest](ctx, r.db, `
		UPDATE task_guests
		SET status = 'declined',
		    responded_at = NOW()
		WHERE id = $1
		  AND LOWER(email) = LOWER($2)
		  AND status = 'pending'
		RETURNING id, task_id, email, user_id, role, status, invited_by, created_at, responded_at
		`,
		invitationID,
		email)
}

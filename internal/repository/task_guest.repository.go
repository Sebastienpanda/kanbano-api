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

var (
	ErrTaskGuestInvitationAlreadyPending = errors.New("task_guest: invitation already pending for this email")
	ErrTaskGuestAlreadyHasAccess         = errors.New("task_guest: user already has access to this task")
	ErrTaskGuestInvitationExpired        = errors.New("task_guest: invitation expired")
)

// taskGuestColumns is the column list of models.TaskGuest.
const taskGuestColumns = `id, task_id, email, user_id, role, status, invited_by, created_at, expires_at, responded_at`

// TaskGuestRepository manages external, non-organisation-member invitations
// scoped to a single task. An invitation expires 7 days after it was sent
// (or last resent); once accepted, the guest keeps their access until an
// owner or admin removes it, or they join the organisation.
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

// CreateInvitation invites an email on a task. An expired pending
// invitation for the same email is replaced. Returns
// ErrTaskGuestAlreadyHasAccess if the email belongs to a person of the
// task's organisation or to a guest of the task, and
// ErrTaskGuestInvitationAlreadyPending if an invitation is still pending for
// it.
func (r *TaskGuestRepository) CreateInvitation(ctx context.Context, params TaskGuestInvitationParams) (models.TaskGuest, error) {
	role := "view"
	if params.Role != nil {
		role = *params.Role
	}

	var hasAccess bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM users u
			JOIN tasks t ON t.id = $1
			JOIN columns c ON c.id = t.column_id
			JOIN workspaces w ON w.id = c.workspace_id
			JOIN organisations o ON o.id = w.organisation_id
			WHERE LOWER(u.email) = LOWER($2)
			  AND (
			      o.user_id = u.id
			      OR EXISTS(SELECT 1 FROM organisation_members om
			                WHERE om.organisation_id = o.id AND om.member_id = u.id)
			      OR EXISTS(SELECT 1 FROM task_guests tg
			                WHERE tg.task_id = t.id AND tg.user_id = u.id AND tg.status = 'accepted')
			  )
		)
		`,
		params.TaskID,
		params.Email).Scan(&hasAccess)
	if err != nil {
		return models.TaskGuest{}, err
	}
	if hasAccess {
		return models.TaskGuest{}, ErrTaskGuestAlreadyHasAccess
	}

	guest, err := queryStruct[models.TaskGuest](ctx, r.db, `
		INSERT INTO task_guests (task_id, email, invited_by, role)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id, LOWER(email)) WHERE status = 'pending' DO UPDATE
		SET role       = EXCLUDED.role,
		    invited_by = EXCLUDED.invited_by,
		    created_at = NOW(),
		    expires_at = NOW() + INTERVAL '7 days'
		WHERE task_guests.expires_at <= NOW()
		RETURNING `+taskGuestColumns,
		params.TaskID,
		params.Email,
		params.InvitedBy,
		role)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.TaskGuest{}, ErrTaskGuestInvitationAlreadyPending
	}
	return guest, err
}

// ListSentInvitations returns the pending invitations of the task, expired
// ones included so that they can be resent.
func (r *TaskGuestRepository) ListSentInvitations(ctx context.Context, taskID uuid.UUID, limit, offset int) ([]models.TaskGuest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+taskGuestColumns+`
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

// ListReceivedInvitations returns the pending, unexpired invitations sent to
// the email, on tasks that still exist.
func (r *TaskGuestRepository) ListReceivedInvitations(ctx context.Context, email string, limit, offset int) ([]models.TaskGuest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tg.id, tg.task_id, tg.email, tg.user_id, tg.role, tg.status, tg.invited_by, tg.created_at, tg.expires_at, tg.responded_at
		FROM task_guests tg
		JOIN tasks t ON t.id = tg.task_id AND t.deleted_at IS NULL
		WHERE LOWER(tg.email) = LOWER($1)
		  AND tg.status = 'pending'
		  AND tg.expires_at > NOW()
		ORDER BY tg.created_at DESC
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

// AcceptInvitation accepts a pending invitation sent to the email. Returns
// ErrTaskGuestInvitationExpired if it has expired,
// ErrTaskGuestAlreadyHasAccess if the user is already a guest of the task or
// has joined the task's organisation since the invitation was sent,
// pgx.ErrNoRows if there is no such pending invitation or the task was
// deleted.
func (r *TaskGuestRepository) AcceptInvitation(ctx context.Context, invitationID, userID uuid.UUID, email string) (models.TaskGuest, error) {
	guest, err := queryStruct[models.TaskGuest](ctx, r.db, `
		UPDATE task_guests
		SET status = 'accepted',
		    user_id = $3,
		    responded_at = NOW()
		WHERE id = $1
		  AND LOWER(email) = LOWER($2)
		  AND status = 'pending'
		  AND expires_at > NOW()
		  AND EXISTS(SELECT 1 FROM tasks t WHERE t.id = task_guests.task_id AND t.deleted_at IS NULL)
		  AND NOT `+inTaskOrganisation("task_guests.task_id", "$3")+`
		RETURNING `+taskGuestColumns,
		invitationID,
		email,
		userID)

	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return models.TaskGuest{}, ErrTaskGuestAlreadyHasAccess
	case errors.Is(err, pgx.ErrNoRows):
		return models.TaskGuest{}, r.pendingInvitationError(ctx, invitationID, userID, email)
	}
	return guest, err
}

// inTaskOrganisation is an SQL condition: the user is the owner or a member
// of the organisation of the task.
func inTaskOrganisation(taskID, userID string) string {
	return `EXISTS(
			SELECT 1
			FROM tasks t
			JOIN columns c ON c.id = t.column_id
			JOIN workspaces w ON w.id = c.workspace_id
			JOIN organisations o ON o.id = w.organisation_id
			WHERE t.id = ` + taskID + `
			  AND (o.user_id = ` + userID + `
			       OR EXISTS(SELECT 1 FROM organisation_members om
			                 WHERE om.organisation_id = o.id AND om.member_id = ` + userID + `))
		)`
}

// pendingInvitationError tells apart an expired invitation, one sent to a
// person of the task's organisation, and a missing one, after an update
// matched no acceptable invitation.
func (r *TaskGuestRepository) pendingInvitationError(ctx context.Context, invitationID, userID uuid.UUID, email string) error {
	var expired, inOrganisation bool
	err := r.db.QueryRow(ctx, `
		SELECT expires_at <= NOW(), `+inTaskOrganisation("task_guests.task_id", "$3")+`
		FROM task_guests
		JOIN tasks t ON t.id = task_guests.task_id AND t.deleted_at IS NULL
		WHERE task_guests.id = $1
		  AND LOWER(task_guests.email) = LOWER($2)
		  AND task_guests.status = 'pending'
		`,
		invitationID,
		email,
		userID).Scan(&expired, &inOrganisation)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return pgx.ErrNoRows
	case err != nil:
		return err
	case expired:
		return ErrTaskGuestInvitationExpired
	case inOrganisation:
		return ErrTaskGuestAlreadyHasAccess
	}
	return pgx.ErrNoRows
}

func (r *TaskGuestRepository) DeclineInvitation(ctx context.Context, invitationID uuid.UUID, email string) (models.TaskGuest, error) {
	return queryStruct[models.TaskGuest](ctx, r.db, `
		UPDATE task_guests
		SET status = 'declined',
		    responded_at = NOW()
		WHERE id = $1
		  AND LOWER(email) = LOWER($2)
		  AND status = 'pending'
		RETURNING `+taskGuestColumns,
		invitationID,
		email)
}

// SentInvitation is a pending invitation with the workspace of its task.
type SentInvitation struct {
	models.TaskGuest
	WorkspaceID uuid.UUID
}

// GetSentInvitation returns a pending invitation sent by the user, expired
// or not, on a task that still exists. Returns pgx.ErrNoRows otherwise.
func (r *TaskGuestRepository) GetSentInvitation(ctx context.Context, invitationID, invitedBy uuid.UUID) (SentInvitation, error) {
	var inv SentInvitation
	err := r.db.QueryRow(ctx, `
		SELECT tg.id, tg.task_id, tg.email, tg.user_id, tg.role, tg.status, tg.invited_by, tg.created_at, tg.expires_at, tg.responded_at,
			c.workspace_id
		FROM task_guests tg
		JOIN tasks t ON t.id = tg.task_id AND t.deleted_at IS NULL
		JOIN columns c ON c.id = t.column_id AND c.deleted_at IS NULL
		WHERE tg.id = $1
		  AND tg.invited_by = $2
		  AND tg.status = 'pending'
		`,
		invitationID,
		invitedBy).Scan(
		&inv.ID, &inv.TaskID, &inv.Email, &inv.UserID, &inv.Role, &inv.Status, &inv.InvitedBy,
		&inv.CreatedAt, &inv.ExpiresAt, &inv.RespondedAt, &inv.WorkspaceID)
	return inv, err
}

// Resend gives a pending invitation 7 more days from now. Returns
// pgx.ErrNoRows if it is no longer pending.
func (r *TaskGuestRepository) Resend(ctx context.Context, invitationID uuid.UUID) (models.TaskGuest, error) {
	return queryStruct[models.TaskGuest](ctx, r.db, `
		UPDATE task_guests
		SET expires_at = NOW() + INTERVAL '7 days'
		WHERE id = $1
		  AND status = 'pending'
		RETURNING `+taskGuestColumns,
		invitationID)
}

// SetGuestRole changes the role ('view' or 'edit') of a guest of the task.
// Returns pgx.ErrNoRows if the user is not a guest of the task.
func (r *TaskGuestRepository) SetGuestRole(ctx context.Context, taskID, userID uuid.UUID, role string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE task_guests
		SET role = $3
		WHERE task_id = $1
		  AND user_id = $2
		  AND status = 'accepted'
		`,
		taskID,
		userID,
		role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// RemoveGuest removes a guest's access to the task; they no longer appear
// among its assignees. Returns pgx.ErrNoRows if the user is not a guest of
// the task.
func (r *TaskGuestRepository) RemoveGuest(ctx context.Context, taskID, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM task_guests
		WHERE task_id = $1
		  AND user_id = $2
		  AND status = 'accepted'
		`,
		taskID,
		userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

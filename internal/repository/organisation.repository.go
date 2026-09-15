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

var ErrInvitationAlreadyPending = errors.New("organisation: invitation already pending for this email")

type OrganisationRepository struct {
	db *pgxpool.Pool
}

func NewOrganisationRepository(db *pgxpool.Pool) *OrganisationRepository {
	return &OrganisationRepository{db: db}
}

func (r *OrganisationRepository) GetOrganisationWithMembers(ctx context.Context, userID uuid.UUID) (models.Organisation, error) {
	var org models.Organisation

	row := r.db.QueryRow(ctx, `
		SELECT id, user_id
		FROM organisations
		WHERE user_id = $1
	`, userID)

	if err := row.Scan(&org.ID, &org.UserID); err != nil {
		return models.Organisation{}, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			u.id,
			u.name,
			u.avatar_version,
			om.joined_at,
			om.role
		FROM organisation_members om
		JOIN users u ON u.id = om.member_id
		WHERE om.organisation_id = $1
		ORDER BY om.joined_at
	`, org.ID)
	if err != nil {
		return models.Organisation{}, err
	}

	org.Members, err = pgx.CollectRows(rows, pgx.RowToStructByName[models.OrganisationMember])
	if err != nil {
		return models.Organisation{}, err
	}
	if org.Members == nil {
		org.Members = []models.OrganisationMember{}
	}

	return org, nil
}

type InvitationParams struct {
	OrganisationID uuid.UUID
	Email          string
	InvitedBy      uuid.UUID
	WorkspaceID    *uuid.UUID
	ColumnID       *uuid.UUID
	TaskID         *uuid.UUID
	Role           *string
}

func (r *OrganisationRepository) CreateInvitation(ctx context.Context, params InvitationParams) (models.OrganisationInvitation, error) {
	invitation, err := queryStruct[models.OrganisationInvitation](ctx, r.db, `
		INSERT INTO organisation_invitations (organisation_id, email, invited_by, workspace_id, column_id, task_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, organisation_id, email, invited_by, status, created_at, responded_at, workspace_id, column_id, task_id, role
		`,
		params.OrganisationID,
		params.Email,
		params.InvitedBy,
		params.WorkspaceID,
		params.ColumnID,
		params.TaskID,
		params.Role)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return models.OrganisationInvitation{}, ErrInvitationAlreadyPending
	}
	return invitation, err
}

func (r *OrganisationRepository) ListSentInvitations(ctx context.Context, organisationID uuid.UUID, limit, offset int) ([]models.OrganisationInvitation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, organisation_id, email, invited_by, status, created_at, responded_at, workspace_id, column_id, task_id, role
		FROM organisation_invitations
		WHERE organisation_id = $1
		  AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
		`,
		organisationID,
		limit,
		offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.OrganisationInvitation])
}

func (r *OrganisationRepository) ListReceivedInvitations(ctx context.Context, email string, limit, offset int) ([]models.OrganisationInvitation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, organisation_id, email, invited_by, status, created_at, responded_at, workspace_id, column_id, task_id, role
		FROM organisation_invitations
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
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.OrganisationInvitation])
}

func (r *OrganisationRepository) AcceptInvitation(ctx context.Context, invitationID, userID uuid.UUID, email string) (models.OrganisationInvitation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.OrganisationInvitation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	invitation, err := queryStruct[models.OrganisationInvitation](ctx, tx, `
		UPDATE organisation_invitations
		SET status = 'accepted',
		    responded_at = NOW()
		WHERE id = $1
		  AND LOWER(email) = LOWER($2)
		  AND status = 'pending'
		RETURNING id, organisation_id, email, invited_by, status, created_at, responded_at, workspace_id, column_id, task_id, role
		`,
		invitationID,
		email)
	if err != nil {
		return models.OrganisationInvitation{}, err
	}

	if invitation.WorkspaceID == nil {
		if err := applyOrgMembership(ctx, tx, &invitation, userID); err != nil {
			return models.OrganisationInvitation{}, err
		}
	}

	switch {
	case invitation.WorkspaceID != nil && invitation.TaskID != nil:
		if err := applyTaskAssignment(ctx, tx, &invitation, userID); err != nil {
			return models.OrganisationInvitation{}, err
		}
	case invitation.WorkspaceID != nil:
		if err := applyWorkspaceAccessGrant(ctx, tx, &invitation, userID); err != nil {
			return models.OrganisationInvitation{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.OrganisationInvitation{}, err
	}

	return invitation, nil
}

func invitationRole(invitation *models.OrganisationInvitation) string {
	if invitation.Role != nil {
		return *invitation.Role
	}
	return "view"
}

func applyOrgMembership(ctx context.Context, tx pgx.Tx, invitation *models.OrganisationInvitation, userID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO organisation_members (member_id, organisation_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (member_id, organisation_id) DO UPDATE SET role = EXCLUDED.role
		`,
		userID,
		invitation.OrganisationID,
		invitationRole(invitation))
	return err
}

func applyTaskAssignment(ctx context.Context, tx pgx.Tx, invitation *models.OrganisationInvitation, userID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO task_assignees (task_id, member_id, role, assigned_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id, member_id) DO UPDATE SET role = EXCLUDED.role
		`,
		*invitation.TaskID,
		userID,
		invitationRole(invitation),
		invitation.InvitedBy)
	return err
}

func applyWorkspaceAccessGrant(ctx context.Context, tx pgx.Tx, invitation *models.OrganisationInvitation, userID uuid.UUID) error {
	insertAccessGrant := `
		INSERT INTO access_grants (workspace_id, column_id, member_id, role, granted_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (member_id, workspace_id) WHERE column_id IS NULL DO UPDATE SET role = EXCLUDED.role
		`
	if invitation.ColumnID != nil {
		insertAccessGrant = `
		INSERT INTO access_grants (workspace_id, column_id, member_id, role, granted_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (member_id, column_id) WHERE column_id IS NOT NULL DO UPDATE SET role = EXCLUDED.role
		`
	}

	_, err := tx.Exec(ctx, insertAccessGrant,
		*invitation.WorkspaceID,
		invitation.ColumnID,
		userID,
		invitationRole(invitation),
		invitation.InvitedBy)
	return err
}

func (r *OrganisationRepository) DeclineInvitation(ctx context.Context, invitationID uuid.UUID, email string) (models.OrganisationInvitation, error) {
	return queryStruct[models.OrganisationInvitation](ctx, r.db, `
		UPDATE organisation_invitations
		SET status = 'declined',
		    responded_at = NOW()
		WHERE id = $1
		  AND LOWER(email) = LOWER($2)
		  AND status = 'pending'
		RETURNING id, organisation_id, email, invited_by, status, created_at, responded_at, workspace_id, column_id, task_id, role
		`,
		invitationID,
		email)
}

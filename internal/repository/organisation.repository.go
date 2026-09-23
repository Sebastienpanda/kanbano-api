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

// GetMemberRole returns the caller's role within their organisation: 'edit'
// if they own it, otherwise their organisation_members.role. Returns
// pgx.ErrNoRows if the user neither owns nor belongs to an organisation.
func (r *OrganisationRepository) GetMemberRole(ctx context.Context, userID uuid.UUID) (string, error) {
	var role *string
	row := r.db.QueryRow(ctx, `
		SELECT CASE
			WHEN EXISTS(SELECT 1 FROM organisations WHERE user_id = $1) THEN 'edit'
			ELSE (
				SELECT om.role FROM organisation_members om
				WHERE om.member_id = $1
				ORDER BY om.joined_at
				LIMIT 1
			)
		END
		`,
		userID)
	if err := row.Scan(&role); err != nil {
		return "", err
	}
	if role == nil {
		return "", pgx.ErrNoRows
	}
	return *role, nil
}

// GetMemberProfile returns memberID's identity, their organisation role, and
// their effective role on every workspace of that organisation. It only
// succeeds if callerID and memberID belong to the same organisation (either
// as owner or as organisation_members); otherwise it returns
// pgx.ErrNoRows. A workspace's role follows the same resolution as
// HasWorkspaceEditAccess: workspace creator or organisation owner is always
// 'edit', a workspace_members override takes priority when present,
// otherwise the organisation role applies.
func (r *OrganisationRepository) GetMemberProfile(ctx context.Context, callerID, memberID uuid.UUID) (models.MemberProfile, error) {
	var profile models.MemberProfile
	row := r.db.QueryRow(ctx, `
		WITH org AS (
			SELECT o.id, o.user_id AS owner_id
			FROM organisations o
			WHERE o.user_id = $1
			   OR EXISTS(SELECT 1 FROM organisation_members om WHERE om.organisation_id = o.id AND om.member_id = $1)
			LIMIT 1
		)
		SELECT u.id, u.email, u.name, u.avatar_version,
			CASE WHEN org.owner_id = u.id THEN 'edit' ELSE om.role END AS role
		FROM org
		JOIN users u ON u.id = $2
		LEFT JOIN organisation_members om ON om.organisation_id = org.id AND om.member_id = $2
		WHERE org.owner_id = $2 OR om.member_id IS NOT NULL
		`,
		callerID,
		memberID)
	if err := row.Scan(&profile.ID, &profile.Email, &profile.Name, &profile.AvatarVersion, &profile.Role); err != nil {
		return models.MemberProfile{}, err
	}

	rows, err := r.db.Query(ctx, `
		WITH org AS (
			SELECT o.id, o.user_id AS owner_id
			FROM organisations o
			WHERE o.user_id = $1
			   OR EXISTS(SELECT 1 FROM organisation_members om WHERE om.organisation_id = o.id AND om.member_id = $1)
			LIMIT 1
		)
		SELECT
			w.id,
			w.name,
			w.created_at,
			COALESCE(
				CASE WHEN w.created_by = $2 THEN 'edit' ELSE wm.role END,
				'view'
			) AS role,
			COALESCE(
				CASE WHEN w.created_by = $2 THEN 'public' ELSE wm.visibility END,
				'private'
			) AS visibility
		FROM workspaces w
		JOIN org ON w.organisation_id = org.id
		LEFT JOIN workspace_members wm ON wm.workspace_id = w.id AND wm.member_id = $2
		WHERE w.deleted_at IS NULL
		ORDER BY w.name
		`,
		callerID,
		memberID)
	if err != nil {
		return models.MemberProfile{}, err
	}

	profile.Workspaces, err = pgx.CollectRows(rows, pgx.RowToStructByName[models.WorkspaceRole])
	if err != nil {
		return models.MemberProfile{}, err
	}
	if profile.Workspaces == nil {
		profile.Workspaces = []models.WorkspaceRole{}
	}

	return profile, nil
}

type InvitationParams struct {
	OrganisationID uuid.UUID
	Email          string
	InvitedBy      uuid.UUID
	Role           *string
}

func (r *OrganisationRepository) CreateInvitation(ctx context.Context, params InvitationParams) (models.OrganisationInvitation, error) {
	invitation, err := queryStruct[models.OrganisationInvitation](ctx, r.db, `
		INSERT INTO organisation_invitations (organisation_id, email, invited_by, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, organisation_id, email, invited_by, status, created_at, responded_at, role
		`,
		params.OrganisationID,
		params.Email,
		params.InvitedBy,
		params.Role)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return models.OrganisationInvitation{}, ErrInvitationAlreadyPending
	}
	return invitation, err
}

func (r *OrganisationRepository) ListSentInvitations(ctx context.Context, organisationID uuid.UUID, limit, offset int) ([]models.OrganisationInvitation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, organisation_id, email, invited_by, status, created_at, responded_at, role
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
		SELECT id, organisation_id, email, invited_by, status, created_at, responded_at, role
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
		RETURNING id, organisation_id, email, invited_by, status, created_at, responded_at, role
		`,
		invitationID,
		email)
	if err != nil {
		return models.OrganisationInvitation{}, err
	}

	if err := applyOrgMembership(ctx, tx, &invitation, userID); err != nil {
		return models.OrganisationInvitation{}, err
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

func (r *OrganisationRepository) DeclineInvitation(ctx context.Context, invitationID uuid.UUID, email string) (models.OrganisationInvitation, error) {
	return queryStruct[models.OrganisationInvitation](ctx, r.db, `
		UPDATE organisation_invitations
		SET status = 'declined',
		    responded_at = NOW()
		WHERE id = $1
		  AND LOWER(email) = LOWER($2)
		  AND status = 'pending'
		RETURNING id, organisation_id, email, invited_by, status, created_at, responded_at, role
		`,
		invitationID,
		email)
}

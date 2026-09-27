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
	ErrInvitationAlreadyPending  = errors.New("organisation: invitation already pending for this email")
	ErrAlreadyOrganisationMember = errors.New("organisation: user already belongs to this organisation")
)

// OrganisationRepository manages organisations, their members and the
// invitations to join them. A user owns at most one organisation
// (organisations.user_id, never stored in organisation_members) and may be
// admin or member of others. A soft-deleted organisation no longer exists
// for anyone. The caller checks the permissions.
type OrganisationRepository struct {
	db *pgxpool.Pool
}

func NewOrganisationRepository(db *pgxpool.Pool) *OrganisationRepository {
	return &OrganisationRepository{db: db}
}

// ListForUser returns the organisations the user owns or belongs to, with
// their role in each; the owned one first.
func (r *OrganisationRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]models.OrganisationSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT o.id, o.name, o.user_id,
			CASE WHEN o.user_id = $1 THEN 'owner' ELSE om.role END AS role
		FROM organisations o
		LEFT JOIN organisation_members om
		       ON om.organisation_id = o.id AND om.member_id = $1
		WHERE o.deleted_at IS NULL
		  AND (o.user_id = $1 OR om.member_id IS NOT NULL)
		ORDER BY o.user_id <> $1, o.name
		`,
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.OrganisationSummary])
}

// GetRole returns the user's role in the organisation: 'owner', 'admin' or
// 'member'. Returns pgx.ErrNoRows if the organisation does not exist or the
// user does not belong to it.
func (r *OrganisationRepository) GetRole(ctx context.Context, organisationID, userID uuid.UUID) (string, error) {
	var role string
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN o.user_id = $2 THEN 'owner' ELSE om.role END
		FROM organisations o
		LEFT JOIN organisation_members om
		       ON om.organisation_id = o.id AND om.member_id = $2
		WHERE o.id = $1
		  AND o.deleted_at IS NULL
		  AND (o.user_id = $2 OR om.member_id IS NOT NULL)
		`,
		organisationID,
		userID).Scan(&role)
	return role, err
}

// OwnedID returns the id of the organisation the user owns. Returns
// pgx.ErrNoRows if they own none (or deleted it).
func (r *OrganisationRepository) OwnedID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT id
		FROM organisations
		WHERE user_id = $1
		  AND deleted_at IS NULL
		`,
		userID).Scan(&id)
	return id, err
}

// Get returns the organisation with its members, the owner first. Returns
// pgx.ErrNoRows if the organisation does not exist.
func (r *OrganisationRepository) Get(ctx context.Context, organisationID uuid.UUID) (models.Organisation, error) {
	var org models.Organisation
	err := r.db.QueryRow(ctx, `
		SELECT id, name, user_id
		FROM organisations
		WHERE id = $1
		  AND deleted_at IS NULL
		`,
		organisationID).Scan(&org.ID, &org.Name, &org.UserID)
	if err != nil {
		return models.Organisation{}, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, name, avatar_version, joined_at, role
		FROM (
			SELECT u.id, u.name, u.avatar_version, NULL::timestamptz AS joined_at, 'owner' AS role, 0 AS rank
			FROM organisations o
			JOIN users u ON u.id = o.user_id
			WHERE o.id = $1
			UNION ALL
			SELECT u.id, u.name, u.avatar_version, om.joined_at, om.role, 1
			FROM organisation_members om
			JOIN users u ON u.id = om.member_id
			WHERE om.organisation_id = $1
		) AS m
		ORDER BY rank, joined_at NULLS FIRST
		`,
		organisationID)
	if err != nil {
		return models.Organisation{}, err
	}

	org.Members, err = pgx.CollectRows(rows, pgx.RowToStructByName[models.OrganisationMember])
	if err != nil {
		return models.Organisation{}, err
	}
	return org, nil
}

// Rename renames the organisation. Returns pgx.ErrNoRows if it does not
// exist.
func (r *OrganisationRepository) Rename(ctx context.Context, organisationID uuid.UUID, name string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE organisations
		SET name = $2
		WHERE id = $1
		  AND deleted_at IS NULL
		`,
		organisationID,
		name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// SoftDelete soft-deletes the organisation along with its workspaces,
// columns and tasks. Its members lose every access to it. Returns
// pgx.ErrNoRows if it does not exist.
func (r *OrganisationRepository) SoftDelete(ctx context.Context, organisationID, userID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE organisations
		SET deleted_at = NOW(),
		    deleted_by = $2
		WHERE id = $1
		  AND deleted_at IS NULL
		`,
		organisationID,
		userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	_, err = tx.Exec(ctx, `
		UPDATE tasks
		SET deleted_at = NOW(),
		    deleted_by = $2
		WHERE deleted_at IS NULL
		  AND column_id IN (
		      SELECT c.id
		      FROM columns c
		      JOIN workspaces w ON w.id = c.workspace_id
		      WHERE w.organisation_id = $1
		  )
		`,
		organisationID,
		userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE columns
		SET deleted_at = NOW(),
		    deleted_by = $2
		WHERE deleted_at IS NULL
		  AND workspace_id IN (SELECT id FROM workspaces WHERE organisation_id = $1)
		`,
		organisationID,
		userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE workspaces
		SET deleted_at = NOW(),
		    deleted_by = $2
		WHERE organisation_id = $1
		  AND deleted_at IS NULL
		`,
		organisationID,
		userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// SetMemberRole changes the organisation role ('admin' or 'member') of a
// member. Returns pgx.ErrNoRows if the user is not a member of the
// organisation (the owner is not one).
func (r *OrganisationRepository) SetMemberRole(ctx context.Context, organisationID, memberID uuid.UUID, role string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE organisation_members om
		SET role = $3
		FROM organisations o
		WHERE o.id = om.organisation_id
		  AND o.deleted_at IS NULL
		  AND om.organisation_id = $1
		  AND om.member_id = $2
		`,
		organisationID,
		memberID,
		role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// RemoveMember removes a member from the organisation, along with their
// access to its workspaces and their assignments on its tasks, so that
// nothing comes back if they are invited again. Returns pgx.ErrNoRows if the
// user is not a member of the organisation (the owner is not one).
func (r *OrganisationRepository) RemoveMember(ctx context.Context, organisationID, memberID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		DELETE FROM organisation_members om
		USING organisations o
		WHERE o.id = om.organisation_id
		  AND o.deleted_at IS NULL
		  AND om.organisation_id = $1
		  AND om.member_id = $2
		`,
		organisationID,
		memberID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM workspace_members wm
		USING workspaces w
		WHERE w.id = wm.workspace_id
		  AND w.organisation_id = $1
		  AND wm.member_id = $2
		`,
		organisationID,
		memberID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM task_assignees ta
		USING tasks t, columns c, workspaces w
		WHERE t.id = ta.task_id
		  AND c.id = t.column_id
		  AND w.id = c.workspace_id
		  AND w.organisation_id = $1
		  AND ta.member_id = $2
		`,
		organisationID,
		memberID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetMemberProfile returns the identity of a person of the organisation,
// their role in it and their access to its workspaces: every workspace of
// the organisation when the caller is its owner, only the workspaces shared
// with the caller otherwise. Returns pgx.ErrNoRows if the person does not
// belong to the organisation.
func (r *OrganisationRepository) GetMemberProfile(ctx context.Context, organisationID, callerID, memberID uuid.UUID) (models.MemberProfile, error) {
	var profile models.MemberProfile
	row := r.db.QueryRow(ctx, `
		SELECT u.id, u.email, u.name, u.avatar_version,
			CASE WHEN o.user_id = u.id THEN 'owner' ELSE om.role END AS role
		FROM organisations o
		JOIN users u ON u.id = $2
		LEFT JOIN organisation_members om
		       ON om.organisation_id = o.id AND om.member_id = u.id
		WHERE o.id = $1
		  AND o.deleted_at IS NULL
		  AND (o.user_id = u.id OR om.member_id IS NOT NULL)
		`,
		organisationID,
		memberID)
	if err := row.Scan(&profile.ID, &profile.Email, &profile.Name, &profile.AvatarVersion, &profile.Role); err != nil {
		return models.MemberProfile{}, err
	}

	// The owner has 'edit' access to every workspace; a member only has the
	// access of their workspace_members row, 'private' (none) by default.
	rows, err := r.db.Query(ctx, `
		SELECT
			w.id,
			w.name,
			w.created_at,
			CASE WHEN o.user_id = $3 THEN 'edit' ELSE COALESCE(wm.role, 'view') END AS role,
			CASE WHEN o.user_id = $3 THEN 'public' ELSE COALESCE(wm.visibility, 'private') END AS visibility
		FROM workspaces w
		JOIN organisations o ON o.id = w.organisation_id
		JOIN workspace_access caller ON caller.workspace_id = w.id AND caller.user_id = $2
		LEFT JOIN workspace_members wm ON wm.workspace_id = w.id AND wm.member_id = $3
		WHERE w.organisation_id = $1
		ORDER BY w.name
		`,
		organisationID,
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

// CreateInvitation invites an email into the organisation. Returns
// ErrAlreadyOrganisationMember if that email is the owner's or a member's,
// ErrInvitationAlreadyPending if an invitation is already pending for it.
func (r *OrganisationRepository) CreateInvitation(ctx context.Context, params InvitationParams) (models.OrganisationInvitation, error) {
	invitation, err := queryStruct[models.OrganisationInvitation](ctx, r.db, `
		INSERT INTO organisation_invitations (organisation_id, email, invited_by, role)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
			SELECT 1
			FROM users u
			JOIN organisations o ON o.id = $1
			LEFT JOIN organisation_members om
			       ON om.organisation_id = o.id AND om.member_id = u.id
			WHERE LOWER(u.email) = LOWER($2)
			  AND (o.user_id = u.id OR om.member_id IS NOT NULL)
		)
		RETURNING id, organisation_id, email, invited_by, status, created_at, responded_at, role
		`,
		params.OrganisationID,
		params.Email,
		params.InvitedBy,
		params.Role)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.OrganisationInvitation{}, ErrAlreadyOrganisationMember
	}
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
		SELECT i.id, i.organisation_id, i.email, i.invited_by, i.status, i.created_at, i.responded_at, i.role
		FROM organisation_invitations i
		JOIN organisations o ON o.id = i.organisation_id AND o.deleted_at IS NULL
		WHERE LOWER(i.email) = LOWER($1)
		  AND i.status = 'pending'
		ORDER BY i.created_at DESC
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

// AcceptInvitation accepts the invitation and makes the user a member of
// the organisation. If they were a guest on tasks of the organisation, those
// guest accesses become access to the workspaces of the tasks, with the
// guest role ('edit' if they had it on one task of the workspace). Returns
// pgx.ErrNoRows if the invitation is not pending for this email, or the
// organisation no longer exists or is the user's own.
func (r *OrganisationRepository) AcceptInvitation(ctx context.Context, invitationID, userID uuid.UUID, email string) (models.OrganisationInvitation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.OrganisationInvitation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	invitation, err := queryStruct[models.OrganisationInvitation](ctx, tx, `
		UPDATE organisation_invitations i
		SET status = 'accepted',
		    responded_at = NOW()
		FROM organisations o
		WHERE o.id = i.organisation_id
		  AND o.deleted_at IS NULL
		  AND o.user_id <> $3
		  AND i.id = $1
		  AND LOWER(i.email) = LOWER($2)
		  AND i.status = 'pending'
		RETURNING i.id, i.organisation_id, i.email, i.invited_by, i.status, i.created_at, i.responded_at, i.role
		`,
		invitationID,
		email,
		userID)
	if err != nil {
		return models.OrganisationInvitation{}, err
	}

	if err := applyOrgMembership(ctx, tx, &invitation, userID); err != nil {
		return models.OrganisationInvitation{}, err
	}
	if err := convertGuestAccess(ctx, tx, invitation.OrganisationID, userID); err != nil {
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
	return "member"
}

func applyOrgMembership(ctx context.Context, tx pgx.Tx, invitation *models.OrganisationInvitation, userID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO organisation_members (member_id, organisation_id, role, joined_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (member_id, organisation_id) DO UPDATE SET role = EXCLUDED.role
		`,
		userID,
		invitation.OrganisationID,
		invitationRole(invitation))
	return err
}

// convertGuestAccess removes the user's guest accesses on the tasks of the
// organisation and shares the workspaces of those tasks with them instead,
// with 'edit' if they were guest with 'edit' on at least one task of the
// workspace, 'view' otherwise.
func convertGuestAccess(ctx context.Context, tx pgx.Tx, organisationID, userID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		WITH removed AS (
			DELETE FROM task_guests tg
			USING tasks t, columns c, workspaces w
			WHERE t.id = tg.task_id
			  AND c.id = t.column_id
			  AND w.id = c.workspace_id
			  AND w.organisation_id = $1
			  AND tg.user_id = $2
			  AND tg.status = 'accepted'
			RETURNING w.id AS workspace_id,
			          tg.role,
			          t.deleted_at IS NULL AND c.deleted_at IS NULL AND w.deleted_at IS NULL AS alive
		)
		INSERT INTO workspace_members (workspace_id, member_id, visibility, role)
		SELECT workspace_id, $2, 'public',
			CASE WHEN bool_or(role = 'edit') THEN 'edit' ELSE 'view' END
		FROM removed
		WHERE alive
		GROUP BY workspace_id
		ON CONFLICT (workspace_id, member_id) DO UPDATE
		SET visibility = 'public',
		    role       = EXCLUDED.role
		`,
		organisationID,
		userID)
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

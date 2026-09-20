package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleRepository resolves a member's effective role. Access is granted at
// exactly two levels: an organisation-wide base role (organisation_members,
// or 'edit' for the workspace creator / organisation owner), and a
// task-specific override (task_assignees). When a task-scoped assignment
// exists for a member, it always wins over the organisation role, even if
// it is less permissive (e.g. 'view' on a task overrides an org-wide
// 'edit').
type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

// HasWorkspaceAccess reports whether the member has any organisation-level
// access to the workspace (creator, organisation owner, or organisation
// member, regardless of role).
func (r *RoleRepository) HasWorkspaceAccess(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspaces w WHERE w.id = $1 AND w.created_by = $2
		) OR EXISTS(
			SELECT 1 FROM workspaces w
			JOIN organisations o ON o.id = w.organisation_id
			WHERE w.id = $1 AND o.user_id = $2
		) OR EXISTS(
			SELECT 1 FROM organisation_members om
			JOIN workspaces w ON w.organisation_id = om.organisation_id
			WHERE w.id = $1 AND om.member_id = $2
		)
		`,
		workspaceID,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

// HasWorkspaceEditAccess reports whether the member has edit access to the
// workspace. The workspace creator and the organisation owner always have
// edit access. For everyone else, a workspace_members override — when
// present — is authoritative, even if it is less permissive than the
// organisation role; without an override, the organisation-level role
// applies.
func (r *RoleRepository) HasWorkspaceEditAccess(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT
			EXISTS(SELECT 1 FROM workspaces w WHERE w.id = $1 AND w.created_by = $2)
			OR EXISTS(
				SELECT 1 FROM workspaces w
				JOIN organisations o ON o.id = w.organisation_id
				WHERE w.id = $1 AND o.user_id = $2
			)
			OR COALESCE(
				(SELECT wm.role = 'edit' FROM workspace_members wm WHERE wm.workspace_id = $1 AND wm.member_id = $2),
				EXISTS(
					SELECT 1 FROM organisation_members om
					JOIN workspaces w ON w.organisation_id = om.organisation_id
					WHERE w.id = $1 AND om.member_id = $2 AND om.role = 'edit'
				)
			)
		`,
		workspaceID,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

// HasTaskAccess reports whether the member can see the given task: either
// through a task-specific assignment, or through organisation-level access
// to the workspace it belongs to.
func (r *RoleRepository) HasTaskAccess(ctx context.Context, workspaceID, taskID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM task_assignees ta WHERE ta.task_id = $2 AND ta.member_id = $3
		) OR EXISTS(
			SELECT 1 FROM workspaces w WHERE w.id = $1 AND w.created_by = $3
		) OR EXISTS(
			SELECT 1 FROM workspaces w
			JOIN organisations o ON o.id = w.organisation_id
			WHERE w.id = $1 AND o.user_id = $3
		) OR EXISTS(
			SELECT 1 FROM organisation_members om
			JOIN workspaces w ON w.organisation_id = om.organisation_id
			WHERE w.id = $1 AND om.member_id = $3
		)
		`,
		workspaceID,
		taskID,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

// HasTaskEditAccess reports whether the member can edit the given task. A
// task-specific assignment, when present, is authoritative: its role alone
// decides, even if the member's organisation role would otherwise grant (or
// deny) edit access. Without an assignment, the organisation-level edit
// role applies.
func (r *RoleRepository) HasTaskEditAccess(ctx context.Context, workspaceID, taskID, userID uuid.UUID) (bool, error) {
	var canEdit bool
	row := r.db.QueryRow(ctx, `
		SELECT CASE
			WHEN EXISTS(SELECT 1 FROM task_assignees ta WHERE ta.task_id = $2 AND ta.member_id = $3)
				THEN EXISTS(
					SELECT 1 FROM task_assignees ta
					WHERE ta.task_id = $2 AND ta.member_id = $3 AND ta.role = 'edit'
				)
			ELSE (
				EXISTS(SELECT 1 FROM workspaces w WHERE w.id = $1 AND w.created_by = $3)
				OR EXISTS(
					SELECT 1 FROM workspaces w
					JOIN organisations o ON o.id = w.organisation_id
					WHERE w.id = $1 AND o.user_id = $3
				)
				OR EXISTS(
					SELECT 1 FROM organisation_members om
					JOIN workspaces w ON w.organisation_id = om.organisation_id
					WHERE w.id = $1 AND om.member_id = $3 AND om.role = 'edit'
				)
			)
		END
		`,
		workspaceID,
		taskID,
		userID)
	err := row.Scan(&canEdit)
	return canEdit, err
}

// SetWorkspaceMemberRole overrides a member's role on a single workspace,
// independently of their organisation-wide role. Only callers with
// organisation-level 'edit' (manage) should be allowed to call this — that
// check belongs to the handler, not here.
func (r *RoleRepository) SetWorkspaceMemberRole(ctx context.Context, workspaceID, memberID uuid.UUID, role string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, member_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (workspace_id, member_id) DO UPDATE SET role = EXCLUDED.role
		`,
		workspaceID,
		memberID,
		role)
	return err
}

// ResolveOrgRole returns the member's organisation role: 'edit' if they own
// it, otherwise their organisation_members.role. Returns pgx.ErrNoRows if
// the user neither owns nor belongs to an organisation.
func (r *RoleRepository) ResolveOrgRole(ctx context.Context, userID uuid.UUID) (string, error) {
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

// ResolveTaskRole returns the member's effective role on a task ('edit' or
// 'view'), applying the task_assignees-first-then-organisation-role
// resolution, along with whether they have any access to it at all.
func (r *RoleRepository) ResolveTaskRole(ctx context.Context, workspaceID, taskID, userID uuid.UUID) (role string, hasAccess bool, err error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			(
				EXISTS(SELECT 1 FROM task_assignees ta WHERE ta.task_id = $2 AND ta.member_id = $3)
				OR EXISTS(SELECT 1 FROM workspaces w WHERE w.id = $1 AND w.created_by = $3)
				OR EXISTS(
					SELECT 1 FROM workspaces w
					JOIN organisations o ON o.id = w.organisation_id
					WHERE w.id = $1 AND o.user_id = $3
				)
				OR EXISTS(
					SELECT 1 FROM organisation_members om
					JOIN workspaces w ON w.organisation_id = om.organisation_id
					WHERE w.id = $1 AND om.member_id = $3
				)
			),
			COALESCE(
				(SELECT ta.role FROM task_assignees ta WHERE ta.task_id = $2 AND ta.member_id = $3),
				CASE
					WHEN EXISTS(SELECT 1 FROM workspaces w WHERE w.id = $1 AND w.created_by = $3) THEN 'edit'
					WHEN EXISTS(
						SELECT 1 FROM workspaces w
						JOIN organisations o ON o.id = w.organisation_id
						WHERE w.id = $1 AND o.user_id = $3
					) THEN 'edit'
					ELSE (
						SELECT om.role FROM organisation_members om
						JOIN workspaces w ON w.organisation_id = om.organisation_id
						WHERE w.id = $1 AND om.member_id = $3
					)
				END,
				'view'
			)
		FROM workspaces w
		WHERE w.id = $1
		`,
		workspaceID,
		taskID,
		userID)
	err = row.Scan(&hasAccess, &role)
	return role, hasAccess, err
}

// BuildWorkspaceAbilities returns the caller's effective permissions on a
// workspace as CASL rules, ready to feed @casl/ability's
// createMongoAbility(rules) on the Angular side. The workspace-level role
// ('edit' → manage, 'view' → read) applies to the Workspace, Column and Task
// subjects. Task-specific assignments (task_assignees) that diverge from
// that baseline are then layered on top: a task downgraded to 'view' gets an
// inverted "cannot manage" rule plus an explicit "can read" rule for that
// task id, and a task upgraded to 'edit' gets an explicit "can manage" rule.
//
// A member with no organisation-level access at all falls back to the
// task_guests source: if they have any accepted guest invitation on a task
// in this workspace, they get read-only access to the whole workspace plus
// an explicit "can manage" rule for each task they were granted 'edit' on.
// This is a distinct source from task_assignees — task_guests is for people
// with no organisation membership, so there is no base org role to override
// in the first place.
//
// hasAccess is false, with rules nil, if the member has no access to the
// workspace at all, through either source.
func (r *RoleRepository) BuildWorkspaceAbilities(ctx context.Context, workspaceID, userID uuid.UUID) (rules []models.AbilityRule, hasAccess bool, err error) {
	hasAccess, err = r.HasWorkspaceAccess(ctx, workspaceID, userID)
	if err != nil {
		return nil, false, err
	}

	if !hasAccess {
		return r.buildGuestAbilities(ctx, workspaceID, userID)
	}

	hasEditAccess, err := r.HasWorkspaceEditAccess(ctx, workspaceID, userID)
	if err != nil {
		return nil, true, err
	}

	baseAction := "read"
	if hasEditAccess {
		baseAction = "manage"
	}

	rules = []models.AbilityRule{
		{Action: baseAction, Subject: "Workspace", Conditions: map[string]any{"id": workspaceID}},
		{Action: baseAction, Subject: "Column", Conditions: map[string]any{"workspace_id": workspaceID}},
		{Action: baseAction, Subject: "Task", Conditions: map[string]any{"workspace_id": workspaceID}},
	}

	taskRoles, err := r.ListTaskRoles(ctx, workspaceID, userID)
	if err != nil {
		return nil, true, err
	}

	for _, tr := range taskRoles {
		switch {
		case hasEditAccess && tr.Role == "view":
			rules = append(rules,
				models.AbilityRule{Action: "manage", Subject: "Task", Conditions: map[string]any{"id": tr.TaskID}, Inverted: true},
				models.AbilityRule{Action: "read", Subject: "Task", Conditions: map[string]any{"id": tr.TaskID}},
			)
		case !hasEditAccess && tr.Role == "edit":
			rules = append(rules,
				models.AbilityRule{Action: "manage", Subject: "Task", Conditions: map[string]any{"id": tr.TaskID}},
			)
		}
	}

	return rules, true, nil
}

func (r *RoleRepository) buildGuestAbilities(ctx context.Context, workspaceID, userID uuid.UUID) ([]models.AbilityRule, bool, error) {
	guestRoles, err := r.ListGuestTaskRoles(ctx, workspaceID, userID)
	if err != nil {
		return nil, false, err
	}
	if len(guestRoles) == 0 {
		return nil, false, nil
	}

	rules := []models.AbilityRule{
		{Action: "read", Subject: "Workspace", Conditions: map[string]any{"id": workspaceID}},
		{Action: "read", Subject: "Column", Conditions: map[string]any{"workspace_id": workspaceID}},
		{Action: "read", Subject: "Task", Conditions: map[string]any{"workspace_id": workspaceID}},
	}
	for _, tr := range guestRoles {
		if tr.Role == "edit" {
			rules = append(rules, models.AbilityRule{Action: "manage", Subject: "Task", Conditions: map[string]any{"id": tr.TaskID}})
		}
	}

	return rules, true, nil
}

// ListGuestTaskRoles returns the member's accepted task_guests role for
// every task in the workspace they have been invited to as an external
// guest (no organisation membership).
func (r *RoleRepository) ListGuestTaskRoles(ctx context.Context, workspaceID, userID uuid.UUID) ([]models.TaskRole, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id AS task_id, t.column_id, tg.role
		FROM task_guests tg
		JOIN tasks t ON t.id = tg.task_id AND t.deleted_at IS NULL
		JOIN columns c ON c.id = t.column_id AND c.deleted_at IS NULL
		WHERE c.workspace_id = $1
		  AND tg.user_id = $2
		  AND tg.status = 'accepted'
		`,
		workspaceID,
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.TaskRole])
}

// ListTaskRoles returns, for every task in the workspace the member can see,
// their effective role. A single query joins task_assignees against the
// member's organisation role and coalesces between the two, so a
// task-specific assignment always takes precedence over the organisation
// role. Tasks the member has no access to at all (no assignment, and no
// organisation membership) are excluded — including the case where the
// member has only been assigned to a subset of the workspace's tasks.
func (r *RoleRepository) ListTaskRoles(ctx context.Context, workspaceID, userID uuid.UUID) ([]models.TaskRole, error) {
	rows, err := r.db.Query(ctx, `
		WITH org_access AS (
			SELECT
				w.id AS workspace_id,
				(
					w.created_by = $2
					OR EXISTS(
						SELECT 1 FROM organisations o
						WHERE o.id = w.organisation_id AND o.user_id = $2
					)
				) AS is_owner,
				(
					SELECT om.role FROM organisation_members om
					WHERE om.organisation_id = w.organisation_id AND om.member_id = $2
				) AS org_role
			FROM workspaces w
			WHERE w.id = $1
		)
		SELECT
			t.id AS task_id,
			t.column_id,
			COALESCE(ta.role, CASE WHEN oa.is_owner THEN 'edit' ELSE oa.org_role END) AS role
		FROM tasks t
		JOIN columns c ON c.id = t.column_id AND c.deleted_at IS NULL
		JOIN org_access oa ON oa.workspace_id = c.workspace_id
		LEFT JOIN task_assignees ta ON ta.task_id = t.id AND ta.member_id = $2
		WHERE t.deleted_at IS NULL
		  AND (ta.member_id IS NOT NULL OR oa.is_owner OR oa.org_role IS NOT NULL)
		ORDER BY c.position, t.position
		`,
		workspaceID,
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.TaskRole])
}

package repository

import (
	"context"
	"kanbano-api/internal/models"
	"kanbano-api/internal/permissions"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleRepository resolves the caller's roles from the workspace_access and
// task_guest_access views (migration 000052), which hold the access rules.
// What those roles allow is decided by the permissions package.
type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

// ResolveAccess returns the user's roles on the workspace, in one query. The
// guest role is resolved for taskID; pass uuid.Nil when no task is involved.
func (r *RoleRepository) ResolveAccess(ctx context.Context, workspaceID, taskID, userID uuid.UUID) (permissions.Access, error) {
	var a permissions.Access
	row := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(wa.org_role, ''),
			COALESCE(wa.workspace_role, ''),
			COALESCE((
				SELECT tga.role FROM task_guest_access tga
				WHERE tga.workspace_id = $1 AND tga.task_id = $2 AND tga.user_id = $3
			), ''),
			EXISTS(
				SELECT 1 FROM task_guest_access tga
				WHERE tga.workspace_id = $1 AND tga.user_id = $3
			)
		FROM (SELECT 1) AS one
		LEFT JOIN workspace_access wa ON wa.workspace_id = $1 AND wa.user_id = $3
		`,
		workspaceID,
		taskID,
		userID)
	err := row.Scan(&a.OrgRole, &a.WorkspaceRole, &a.GuestRole, &a.IsGuest)
	return a, err
}

// TaskInWorkspace reports whether the task exists (not deleted) in the
// workspace.
func (r *RoleRepository) TaskInWorkspace(ctx context.Context, taskID, workspaceID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM tasks t
			JOIN columns c ON c.id = t.column_id
			WHERE t.id = $1
			  AND c.workspace_id = $2
			  AND t.deleted_at IS NULL
			  AND c.deleted_at IS NULL
		)
		`,
		taskID,
		workspaceID).Scan(&exists)
	return exists, err
}

// OrgRoles returns the role of each given user in the workspace's
// organisation ('owner', 'admin' or 'member'). Users outside the
// organisation are missing from the map.
func (r *RoleRepository) OrgRoles(ctx context.Context, workspaceID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, CASE WHEN o.user_id = u.id THEN 'owner' ELSE om.role END
		FROM workspaces w
		JOIN organisations o ON o.id = w.organisation_id
		CROSS JOIN unnest($2::uuid[]) AS u(id)
		LEFT JOIN organisation_members om
		       ON om.organisation_id = w.organisation_id AND om.member_id = u.id
		WHERE w.id = $1
		  AND (o.user_id = u.id OR om.member_id IS NOT NULL)
		`,
		workspaceID,
		uuidStrings(userIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make(map[uuid.UUID]string, len(userIDs))
	for rows.Next() {
		var (
			id   uuid.UUID
			role string
		)
		if err := rows.Scan(&id, &role); err != nil {
			return nil, err
		}
		roles[id] = role
	}
	return roles, rows.Err()
}

// ListTaskRoles returns the tasks of the workspace the caller sees, with
// their role on each: every task with the same role for a member ('edit'
// if they may edit the content), only the tasks shared with them for a
// guest, with their guest role.
func (r *RoleRepository) ListTaskRoles(ctx context.Context, workspaceID, userID uuid.UUID, a permissions.Access) ([]models.TaskRole, error) {
	memberRole := ""
	if permissions.CanViewWorkspace(a) {
		memberRole = permissions.View
		if permissions.CanEditContent(a) {
			memberRole = permissions.Edit
		}
	}

	rows, err := r.db.Query(ctx, `
		SELECT t.id AS task_id, t.column_id, COALESCE(NULLIF($3, ''), tga.role) AS role
		FROM tasks t
		JOIN columns c ON c.id = t.column_id AND c.deleted_at IS NULL
		LEFT JOIN task_guest_access tga ON tga.task_id = t.id AND tga.user_id = $2
		WHERE c.workspace_id = $1
		  AND t.deleted_at IS NULL
		  AND ($3 <> '' OR tga.task_id IS NOT NULL)
		ORDER BY c.position, t.position
		`,
		workspaceID,
		userID,
		memberRole)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.TaskRole])
}

// BuildWorkspaceAbilities returns the caller's permissions on the workspace
// as CASL rules, ready to feed @casl/ability's createMongoAbility(rules) on
// the Angular side. Actions: read, create, update, delete, plus share (manage
// the members' access) on Workspace, and assign and invite (guests) on Task.
// A guest only gets rules on the tasks shared with them.
func (r *RoleRepository) BuildWorkspaceAbilities(ctx context.Context, workspaceID, userID uuid.UUID, a permissions.Access) ([]models.AbilityRule, error) {
	workspace := map[string]any{"id": workspaceID}
	content := map[string]any{"workspace_id": workspaceID}

	if !permissions.CanViewWorkspace(a) {
		return r.buildGuestAbilities(ctx, workspaceID, userID, a)
	}

	rules := []models.AbilityRule{
		{Action: "read", Subject: "Workspace", Conditions: workspace},
		{Action: "read", Subject: "Column", Conditions: content},
		{Action: "read", Subject: "Task", Conditions: content},
	}
	add := func(allowed bool, subject string, conditions map[string]any, actions ...string) {
		if !allowed {
			return
		}
		for _, action := range actions {
			rules = append(rules, models.AbilityRule{Action: action, Subject: subject, Conditions: conditions})
		}
	}
	add(permissions.CanEditWorkspace(a), "Workspace", workspace, "update", "share")
	add(permissions.CanDeleteWorkspace(a), "Workspace", workspace, "delete")
	add(permissions.CanEditContent(a), "Column", content, "update")
	add(permissions.CanCreateOrDeleteContent(a), "Column", content, "create", "delete")
	add(permissions.CanEditContent(a), "Task", content, "update")
	add(permissions.CanCreateOrDeleteContent(a), "Task", content, "create", "delete")
	add(permissions.CanAssign(a), "Task", content, "assign")
	add(permissions.CanInviteGuest(a), "Task", content, "invite")

	return rules, nil
}

func (r *RoleRepository) buildGuestAbilities(ctx context.Context, workspaceID, userID uuid.UUID, a permissions.Access) ([]models.AbilityRule, error) {
	taskRoles, err := r.ListTaskRoles(ctx, workspaceID, userID, a)
	if err != nil {
		return nil, err
	}

	rules := []models.AbilityRule{
		{Action: "read", Subject: "Workspace", Conditions: map[string]any{"id": workspaceID}},
	}
	for _, tr := range taskRoles {
		task := map[string]any{"id": tr.TaskID}
		rules = append(rules, models.AbilityRule{Action: "read", Subject: "Task", Conditions: task})
		if tr.Role == permissions.Edit {
			rules = append(rules, models.AbilityRule{Action: "update", Subject: "Task", Fields: []string{"description"}, Conditions: task})
		}
	}
	return rules, nil
}

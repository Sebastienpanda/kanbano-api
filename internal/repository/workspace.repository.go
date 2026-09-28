package repository

import (
	"context"
	"kanbano-api/internal/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkspaceRepository struct {
	db *pgxpool.Pool
}

func NewWorkspaceRepository(db *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

// accessibleWorkspaces is the accessible_workspaces CTE: the ids of the
// workspaces user $N sees, as a member or as a guest (see the
// workspace_access and task_guest_access views).
func accessibleWorkspaces(userParam string) string {
	return `accessible_workspaces AS (
			SELECT workspace_id AS id FROM workspace_access WHERE user_id = ` + userParam + `
			UNION
			SELECT workspace_id FROM task_guest_access WHERE user_id = ` + userParam + `
		)`
}

func (r *WorkspaceRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Workspace, error) {
	rows, err := r.db.Query(ctx, `
		WITH `+accessibleWorkspaces("$1")+`
		SELECT
			w.id,
			w.name,
			w.description,
			w.organisation_id,
			w.created_by,
			w.updated_by,
			w.deleted_by,
			w.created_at,
			w.updated_at,
			w.deleted_at
		FROM workspaces w
		JOIN accessible_workspaces aw ON aw.id = w.id
		WHERE w.deleted_at IS NULL
		ORDER BY COALESCE(w.updated_at, w.created_at) DESC
		LIMIT $2 OFFSET $3
		`,
		userID,
		limit,
		offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Workspace])
}

func (r *WorkspaceRepository) ListRecent(ctx context.Context, userID uuid.UUID) ([]models.Workspace, error) {
	rows, err := r.db.Query(ctx, `
		WITH `+accessibleWorkspaces("$1")+`
		SELECT
			w.id,
			w.name,
			w.description,
			w.organisation_id,
			w.created_by,
			w.updated_by,
			w.deleted_by,
			w.created_at,
			w.updated_at,
			w.deleted_at
		FROM workspaces w
		JOIN accessible_workspaces aw ON aw.id = w.id
		WHERE w.deleted_at IS NULL
		ORDER BY COALESCE(w.updated_at, w.created_at) DESC
		LIMIT 6
		`,
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Workspace])
}

func (r *WorkspaceRepository) ListNames(ctx context.Context, userID uuid.UUID) ([]models.WorkspaceName, error) {
	rows, err := r.db.Query(ctx, `
		WITH `+accessibleWorkspaces("$1")+`
		SELECT
			w.id,
			w.name
		FROM workspaces w
		JOIN accessible_workspaces aw ON aw.id = w.id
		WHERE w.deleted_at IS NULL
		ORDER BY COALESCE(w.updated_at, w.created_at) DESC
		`,
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.WorkspaceName])
}

func (r *WorkspaceRepository) Search(ctx context.Context, userID uuid.UUID, query string) ([]models.WorkspaceSearchResult, error) {
	rows, err := r.db.Query(ctx, `
		WITH `+accessibleWorkspaces("$1")+`
		SELECT
			w.id,
			w.name,
			w.description
		FROM workspaces w
		JOIN accessible_workspaces aw ON aw.id = w.id
		WHERE w.deleted_at IS NULL
		  AND (w.name ILIKE $2 OR w.description ILIKE $2)
		ORDER BY COALESCE(w.updated_at, w.created_at) DESC
		`,
		userID,
		"%"+query+"%")
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.WorkspaceSearchResult])
}

// Create creates a workspace in the organisation. The caller checks the
// permissions.
func (r *WorkspaceRepository) Create(ctx context.Context, organisationID uuid.UUID, name string, description *string, userID uuid.UUID) (models.Workspace, error) {
	return queryStruct[models.Workspace](ctx, r.db, `
		INSERT INTO workspaces (name, description, created_by, organisation_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, description, organisation_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		name,
		description,
		userID,
		organisationID)
}

// GetByID returns the workspace with its columns, tasks and members. With
// guestOnly, it only returns the tasks shared with the user as a guest (and
// their columns), and no members.
func (r *WorkspaceRepository) GetByID(ctx context.Context, id, userID uuid.UUID, guestOnly bool) (models.WorkspaceDetail, error) {
	rows, err := r.db.Query(ctx, `
		WITH `+accessibleWorkspaces("$2")+`
		SELECT
			w.id,
			w.organisation_id,
			w.name,
			w.description,
			w.created_at,
			w.updated_at,
			c.id,
			c.name,
			c.position,
			c.workspace_id,
			c.created_by,
			c.created_at,
			c.updated_at,
			t.id,
			t.name,
			t.description,
			t.position,
			t.column_id,
			t.tag_id,
			t.status,
			t.created_by,
			t.created_at,
			t.updated_at,
			g.id,
			g.name,
			g.color,
			au.id,
			au.email,
			au.avatar_version
		FROM workspaces w
		JOIN accessible_workspaces aw ON aw.id = w.id
		LEFT JOIN columns c
		    ON c.workspace_id = w.id
		           AND c.deleted_at IS NULL
		           AND (NOT $3 OR c.id IN (SELECT column_id FROM task_guest_access WHERE user_id = $2))
		LEFT JOIN tasks t
		    ON t.column_id = c.id
		           AND t.deleted_at IS NULL
		           AND (NOT $3 OR t.id IN (SELECT task_id FROM task_guest_access WHERE user_id = $2))
		LEFT JOIN tags g
		    ON g.id = t.tag_id
		LEFT JOIN (
		    SELECT task_id, member_id AS user_id FROM task_assignees
		    UNION
		    SELECT task_id, user_id FROM task_guest_access
		) ta
		    ON ta.task_id = t.id
		LEFT JOIN users au
		    ON au.id = ta.user_id
		WHERE w.id = $1
		  AND w.deleted_at IS NULL
		ORDER BY c.position,
		         t.position,
		         t.created_at
		    DESC
		`,
		id,
		userID,
		guestOnly)
	if err != nil {
		return models.WorkspaceDetail{}, err
	}
	defer rows.Close()

	acc := newWorkspaceDetailAccumulator()
	for rows.Next() {
		if err := acc.scanRow(rows); err != nil {
			return models.WorkspaceDetail{}, err
		}
	}
	if err := rows.Err(); err != nil {
		return models.WorkspaceDetail{}, err
	}
	if !acc.initialized {
		return models.WorkspaceDetail{}, pgx.ErrNoRows
	}

	detail := acc.detail
	detail.Columns = acc.buildColumnsWithTasks()

	detail.Members = []models.WorkspaceMember{}
	if guestOnly {
		return detail, nil
	}
	detail.Members, err = r.listPublicMembers(ctx, id)
	if err != nil {
		return models.WorkspaceDetail{}, err
	}
	return detail, nil
}

// listPublicMembers returns the admins and members the workspace is shared
// with (the owner, who sees every workspace, is not listed).
func (r *WorkspaceRepository) listPublicMembers(ctx context.Context, workspaceID uuid.UUID) ([]models.WorkspaceMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			u.id,
			u.name,
			u.email,
			u.avatar_version,
			wa.workspace_role,
			wa.org_role
		FROM workspace_access wa
		JOIN users u ON u.id = wa.user_id
		WHERE wa.workspace_id = $1
		  AND wa.org_role <> 'owner'
		ORDER BY u.name NULLS LAST, u.email
		`,
		workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []models.WorkspaceMember{}
	for rows.Next() {
		var m models.WorkspaceMember
		if err := rows.Scan(&m.ID, &m.Name, &m.Email, &m.AvatarVersion, &m.Role, &m.OrganisationRole); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// workspaceDetailAccumulator collects the denormalized rows of the workspace
// detail query (workspace x columns x tasks x tags x assignees) into the
// nested structure expected by models.WorkspaceDetail.
type workspaceDetailAccumulator struct {
	detail       models.WorkspaceDetail
	columnsMap   map[uuid.UUID]*models.ColumnWithTasks
	columnOrder  []uuid.UUID
	taskOrder    map[uuid.UUID][]uuid.UUID
	tasksMap     map[uuid.UUID]*models.TaskWithTag
	assigneeSeen map[uuid.UUID]map[uuid.UUID]bool
	initialized  bool
}

func newWorkspaceDetailAccumulator() *workspaceDetailAccumulator {
	return &workspaceDetailAccumulator{
		columnsMap:   make(map[uuid.UUID]*models.ColumnWithTasks),
		taskOrder:    make(map[uuid.UUID][]uuid.UUID),
		tasksMap:     make(map[uuid.UUID]*models.TaskWithTag),
		assigneeSeen: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

func (a *workspaceDetailAccumulator) scanRow(rows pgx.Rows) error {
	var (
		wsID                       uuid.UUID
		wsOrgID                    uuid.UUID
		wsName                     string
		wsDesc                     *string
		wsCreatedAt                time.Time
		wsUpdatedAt                *time.Time
		colID, colWsID             *uuid.UUID
		colCreatedBy               *uuid.UUID
		colName                    *string
		colPos                     *int
		colCreatedAt, colUpdatedAt *time.Time
		taskID, taskColID          *uuid.UUID
		taskName                   *string
		taskDesc                   *string
		taskPos                    *int
		taskTagID, taskCreatedBy   *uuid.UUID
		taskStatus                 *string
		taskCreatedAt, taskUpdAt   *time.Time
		tagID                      *uuid.UUID
		tagName                    *string
		tagColor                   *string
		assigneeID                 *uuid.UUID
		assigneeEmail              *string
		assigneeAvatarVersion      *string
	)

	err := rows.Scan(
		&wsID, &wsOrgID, &wsName, &wsDesc, &wsCreatedAt, &wsUpdatedAt,
		&colID, &colName, &colPos, &colWsID, &colCreatedBy, &colCreatedAt, &colUpdatedAt,
		&taskID, &taskName, &taskDesc, &taskPos, &taskColID, &taskTagID, &taskStatus, &taskCreatedBy, &taskCreatedAt, &taskUpdAt,
		&tagID, &tagName, &tagColor,
		&assigneeID, &assigneeEmail, &assigneeAvatarVersion,
	)
	if err != nil {
		return err
	}

	if !a.initialized {
		a.detail.ID = wsID
		a.detail.OrganisationID = wsOrgID
		a.detail.Name = wsName
		a.detail.Description = wsDesc
		a.detail.CreatedAt = wsCreatedAt
		a.detail.UpdatedAt = wsUpdatedAt
		a.initialized = true
	}

	if colID == nil {
		return nil
	}

	if _, exists := a.columnsMap[*colID]; !exists {
		col := models.ColumnWithTasks{
			ID:        *colID,
			Name:      derefStr(colName),
			Position:  derefInt(colPos),
			CreatedBy: derefUUID(colCreatedBy),
			CreatedAt: derefTime(colCreatedAt),
			UpdatedAt: colUpdatedAt,
		}
		a.columnsMap[*colID] = &col
		a.columnOrder = append(a.columnOrder, *colID)
	}

	if taskID == nil {
		return nil
	}

	task, exists := a.tasksMap[*taskID]
	if !exists {
		task = &models.TaskWithTag{
			ID:            *taskID,
			Name:          derefStr(taskName),
			Description:   taskDesc,
			Position:      derefInt(taskPos),
			ColumnID:      derefUUID(taskColID),
			TagID:         taskTagID,
			Status:        taskStatus,
			CreatedBy:     derefUUID(taskCreatedBy),
			CreatedAt:     derefTime(taskCreatedAt),
			UpdatedAt:     taskUpdAt,
			AssignedUsers: []models.TaskAssignedUser{},
		}
		if tagID != nil {
			task.Tag = &models.TagName{ID: *tagID, Name: derefStr(tagName), Color: tagColor}
		}
		a.tasksMap[*taskID] = task
		a.taskOrder[*colID] = append(a.taskOrder[*colID], *taskID)
		a.assigneeSeen[*taskID] = make(map[uuid.UUID]bool)
	}

	if assigneeID != nil && !a.assigneeSeen[*taskID][*assigneeID] {
		a.assigneeSeen[*taskID][*assigneeID] = true
		task.AssignedUsers = append(task.AssignedUsers, models.TaskAssignedUser{
			ID:            *assigneeID,
			Email:         derefStr(assigneeEmail),
			AvatarVersion: assigneeAvatarVersion,
		})
	}

	return nil
}

func (a *workspaceDetailAccumulator) buildColumnsWithTasks() []models.ColumnWithTasks {
	columns := make([]models.ColumnWithTasks, 0, len(a.columnOrder))
	for _, colID := range a.columnOrder {
		col := a.columnsMap[colID]
		col.Tasks = make([]models.TaskWithTag, 0, len(a.taskOrder[colID]))
		for _, tID := range a.taskOrder[colID] {
			col.Tasks = append(col.Tasks, *a.tasksMap[tID])
		}
		columns = append(columns, *col)
	}
	return columns
}

// WorkspaceMemberAccess is an access change for one member of a workspace.
// A nil field keeps the member's current value, or falls back to the column
// default ('private' / 'view') when the member had no access row yet.
type WorkspaceMemberAccess struct {
	MemberID   uuid.UUID
	Visibility *string
	Role       *string
}

// Update updates the workspace's name/description and, in the same
// transaction, the access of the given members. The caller checks the
// permissions, and that every member belongs to the workspace's
// organisation. Returns pgx.ErrNoRows if the workspace does not exist.
func (r *WorkspaceRepository) Update(ctx context.Context, id, userID uuid.UUID, name, description *string, members []WorkspaceMemberAccess) (models.Workspace, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Workspace{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var workspace models.Workspace
	if name != nil || description != nil || len(members) == 0 {
		workspace, err = queryStruct[models.Workspace](ctx, tx, `
			UPDATE workspaces
			SET name        = COALESCE($1, name),
			    description = COALESCE($2, description),
			    updated_by  = $4,
			    updated_at  = NOW()
			WHERE id = $3
			  AND deleted_at IS NULL
			RETURNING id, name, description, organisation_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
			`,
			name,
			description,
			id,
			userID)
	} else {
		workspace, err = queryStruct[models.Workspace](ctx, tx, `
			SELECT id, name, description, organisation_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
			FROM workspaces
			WHERE id = $1
			  AND deleted_at IS NULL
			`,
			id)
	}
	if err != nil {
		return models.Workspace{}, err
	}

	if len(members) > 0 {
		if err := setMembersAccess(ctx, tx, id, members); err != nil {
			return models.Workspace{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Workspace{}, err
	}
	return workspace, nil
}

// setMembersAccess creates or updates the workspace_members row of each
// member.
func setMembersAccess(ctx context.Context, tx pgx.Tx, workspaceID uuid.UUID, members []WorkspaceMemberAccess) error {
	memberIDs := make([]string, len(members))
	visibilities := make([]*string, len(members))
	roles := make([]*string, len(members))
	for i, m := range members {
		memberIDs[i] = m.MemberID.String()
		visibilities[i] = m.Visibility
		roles[i] = m.Role
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, member_id, visibility, role)
		SELECT $1, m.member_id, COALESCE(m.visibility, 'private'), COALESCE(m.role, 'view')
		FROM unnest($2::uuid[], $3::text[], $4::text[]) AS m(member_id, visibility, role)
		ON CONFLICT (workspace_id, member_id) DO NOTHING
		`,
		workspaceID,
		memberIDs,
		visibilities,
		roles)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE workspace_members wm
		SET visibility = COALESCE(m.visibility, wm.visibility),
		    role       = COALESCE(m.role, wm.role)
		FROM unnest($2::uuid[], $3::text[], $4::text[]) AS m(member_id, visibility, role)
		WHERE wm.workspace_id = $1
		  AND wm.member_id = m.member_id
		`,
		workspaceID,
		memberIDs,
		visibilities,
		roles)
	return err
}

// SoftDelete soft-deletes the workspace along with its columns and tasks.
// The caller checks the permissions. Returns pgx.ErrNoRows if the workspace
// does not exist.
func (r *WorkspaceRepository) SoftDelete(ctx context.Context, id, userID uuid.UUID) (models.Workspace, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Workspace{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	workspace, err := queryStruct[models.Workspace](ctx, tx, `
		UPDATE workspaces
		SET deleted_at = NOW(),
		    deleted_by = $2
		WHERE id = $1
		  AND deleted_at IS NULL
		RETURNING id, name, description, organisation_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		id,
		userID)
	if err != nil {
		return models.Workspace{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE columns
		SET deleted_at = NOW(),
		    deleted_by = $2
		WHERE workspace_id = $1
		  AND deleted_at IS NULL
		`,
		id,
		userID)
	if err != nil {
		return models.Workspace{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE tasks
		SET deleted_at = NOW(),
		    deleted_by = $2
		WHERE deleted_at IS NULL
		  AND column_id
		          IN (SELECT id FROM columns WHERE workspace_id = $1)
		`,
		id,
		userID)
	if err != nil {
		return models.Workspace{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return models.Workspace{}, err
	}

	return workspace, nil
}

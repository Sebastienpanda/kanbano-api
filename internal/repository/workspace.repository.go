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

func (r *WorkspaceRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Workspace, error) {
	rows, err := r.db.Query(ctx, `
		WITH accessible_workspaces AS (
			SELECT id FROM workspaces WHERE created_by = $1
			UNION
			SELECT workspace_id FROM access_grants WHERE member_id = $1
			UNION
			SELECT w.id FROM workspaces w
			JOIN organisation_members om ON om.organisation_id = w.organisation_id
			WHERE om.member_id = $1
		)
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
		WITH accessible_workspaces AS (
			SELECT id FROM workspaces WHERE created_by = $1
			UNION
			SELECT workspace_id FROM access_grants WHERE member_id = $1
			UNION
			SELECT w.id FROM workspaces w
			JOIN organisation_members om ON om.organisation_id = w.organisation_id
			WHERE om.member_id = $1
		)
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
		WITH accessible_workspaces AS (
			SELECT id FROM workspaces WHERE created_by = $1
			UNION
			SELECT workspace_id FROM access_grants WHERE member_id = $1
			UNION
			SELECT w.id FROM workspaces w
			JOIN organisation_members om ON om.organisation_id = w.organisation_id
			WHERE om.member_id = $1
		)
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
		WITH accessible_workspaces AS (
			SELECT id FROM workspaces WHERE created_by = $1
			UNION
			SELECT workspace_id FROM access_grants WHERE member_id = $1
			UNION
			SELECT w.id FROM workspaces w
			JOIN organisation_members om ON om.organisation_id = w.organisation_id
			WHERE om.member_id = $1
		)
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

func (r *WorkspaceRepository) Create(ctx context.Context, name string, description *string, userID uuid.UUID) (models.Workspace, error) {
	return queryStruct[models.Workspace](ctx, r.db, `
		INSERT INTO workspaces (name, description, created_by, organisation_id)
		VALUES ($1, $2, $3, (SELECT id FROM organisations WHERE user_id = $3))
		RETURNING id, name, description, organisation_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		name,
		description,
		userID)
}

func (r *WorkspaceRepository) GetByID(ctx context.Context, id, userID uuid.UUID) (models.WorkspaceDetail, error) {
	rows, err := r.db.Query(ctx, `
		WITH accessible_workspaces AS (
			SELECT id FROM workspaces WHERE created_by = $2
			UNION
			SELECT workspace_id FROM access_grants WHERE member_id = $2
			UNION
			SELECT w.id FROM workspaces w
			JOIN organisation_members om ON om.organisation_id = w.organisation_id
			WHERE om.member_id = $2
		)
		SELECT
			w.id,
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
		LEFT JOIN tasks t
		    ON t.column_id = c.id
		           AND t.deleted_at IS NULL
		LEFT JOIN tags g
		    ON g.id = t.tag_id
		LEFT JOIN task_assignees ta
		    ON ta.task_id = t.id
		LEFT JOIN users au
		    ON au.id = ta.member_id
		WHERE w.id = $1
		  AND w.deleted_at IS NULL
		ORDER BY c.position,
		         t.position,
		         t.created_at
		    DESC
		`,
		id,
		userID)
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
	return detail, nil
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
		&wsID, &wsName, &wsDesc, &wsCreatedAt, &wsUpdatedAt,
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

func (r *WorkspaceRepository) Update(ctx context.Context, id, userID uuid.UUID, name, description *string) (models.Workspace, error) {
	return queryStruct[models.Workspace](ctx, r.db, `
		UPDATE workspaces
		SET name        = COALESCE($1, name),
		    description = COALESCE($2, description),
		    updated_by  = $4,
		    updated_at  = NOW()
		WHERE id = $3
		  AND created_by = $4
		  AND deleted_at IS NULL
		RETURNING id, name, description, organisation_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at
		`,
		name,
		description,
		id,
		userID)
}

func (r *WorkspaceRepository) Exists(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		WITH accessible_workspaces AS (
			SELECT id FROM workspaces WHERE created_by = $2
			UNION
			SELECT workspace_id FROM access_grants WHERE member_id = $2
			UNION
			SELECT w.id FROM workspaces w
			JOIN organisation_members om ON om.organisation_id = w.organisation_id
			WHERE om.member_id = $2
		)
		SELECT EXISTS(
			SELECT 1
			FROM workspaces w
			JOIN accessible_workspaces aw ON aw.id = w.id
			WHERE w.id = $1
			  AND w.deleted_at IS NULL
		)
		`,
		id,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

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
		  AND created_by = $2
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

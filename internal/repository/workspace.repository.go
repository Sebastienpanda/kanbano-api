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

func (r *WorkspaceRepository) List(ctx context.Context, userID uuid.UUID) ([]models.Workspace, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			name,
			description,
			user_id,
			created_at,
			updated_at,
			deleted_at
		FROM workspaces
		WHERE user_id = $1
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
		`,
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Workspace])
}

func (r *WorkspaceRepository) ListRecent(ctx context.Context, userID uuid.UUID) ([]models.Workspace, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			name,
			description,
			user_id,
			created_at,
			updated_at,
			deleted_at
		FROM workspaces
		WHERE user_id = $1
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
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
		SELECT
			id,
			name
		FROM workspaces
		WHERE user_id = $1
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
		`,
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.WorkspaceName])
}

func (r *WorkspaceRepository) GetIDByName(ctx context.Context, name string, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	row := r.db.QueryRow(ctx, `
		SELECT
			id
		FROM workspaces
		WHERE name = $1
		  AND user_id = $2
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
		`,
		name,
		userID)

	err := row.Scan(&id)
	return id, err
}

func (r *WorkspaceRepository) Create(ctx context.Context, name string, description *string, userID uuid.UUID) (models.Workspace, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO workspaces (name, description, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, user_id, created_at, updated_at, deleted_at
		`,
		name,
		description,
		userID)
	if err != nil {
		return models.Workspace{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Workspace])
}

func (r *WorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (models.WorkspaceDetail, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			w.id, 
			w.name, 
			w.description, 
			w.user_id, 
			w.created_at, 
			w.updated_at,
			c.id, 
			c.name, 
			c.position, 
			c.workspace_id, 
			c.created_at, 
			c.updated_at,
			t.id, 
			t.name, 
			t.description, 
			t.position, 
			t.column_id, 
			t.state_id, 
			t.created_at, 
			t.updated_at,
			s.id, 
			s.name, 
			s.color
		FROM workspaces w
		LEFT JOIN columns c 
		    ON c.workspace_id = w.id 
		           AND c.deleted_at IS NULL
		LEFT JOIN tasks t 
		    ON t.column_id = c.id 
		           AND t.deleted_at IS NULL
		LEFT JOIN states s 
		    ON s.id = t.state_id
		WHERE w.id = $1 
		  AND w.user_id = $2 
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

	var detail models.WorkspaceDetail
	columnsMap := make(map[uuid.UUID]*models.ColumnWithTasks)
	var columnOrder []uuid.UUID
	initialized := false

	for rows.Next() {
		var (
			wsID, wsUserID             uuid.UUID
			wsName                     string
			wsDesc                     *string
			wsCreatedAt                time.Time
			wsUpdatedAt                *time.Time
			colID, colWsID             *uuid.UUID
			colName                    *string
			colPos                     *int
			colCreatedAt, colUpdatedAt *time.Time
			taskID, taskColID          *uuid.UUID
			taskName                   *string
			taskDesc                   *string
			taskPos                    *int
			taskStateID                *uuid.UUID
			taskCreatedAt, taskUpdAt   *time.Time
			stateID                    *uuid.UUID
			stateName                  *string
			stateColor                 *string
		)

		err := rows.Scan(
			&wsID, &wsName, &wsDesc, &wsUserID, &wsCreatedAt, &wsUpdatedAt,
			&colID, &colName, &colPos, &colWsID, &colCreatedAt, &colUpdatedAt,
			&taskID, &taskName, &taskDesc, &taskPos, &taskColID, &taskStateID, &taskCreatedAt, &taskUpdAt,
			&stateID, &stateName, &stateColor,
		)
		if err != nil {
			return models.WorkspaceDetail{}, err
		}

		if !initialized {
			detail.Workspace = models.Workspace{
				ID:          wsID,
				Name:        wsName,
				Description: wsDesc,
				UserID:      wsUserID,
				CreatedAt:   wsCreatedAt,
				UpdatedAt:   wsUpdatedAt,
			}
			initialized = true
		}

		if colID == nil {
			continue
		}

		if _, exists := columnsMap[*colID]; !exists {
			col := models.ColumnWithTasks{
				Column: models.Column{
					ID:          *colID,
					Name:        derefStr(colName),
					Position:    derefInt(colPos),
					WorkspaceID: derefUUID(colWsID),
					CreatedAt:   derefTime(colCreatedAt),
					UpdatedAt:   derefTime(colUpdatedAt),
				},
				Tasks: []models.TaskWithState{},
			}
			columnsMap[*colID] = &col
			columnOrder = append(columnOrder, *colID)
		}

		if taskID != nil {
			task := models.TaskWithState{
				Task: models.Task{
					ID:          *taskID,
					Name:        derefStr(taskName),
					Description: taskDesc,
					Position:    derefInt(taskPos),
					ColumnID:    derefUUID(taskColID),
					StateID:     taskStateID,
					CreatedAt:   derefTime(taskCreatedAt),
					UpdatedAt:   derefTime(taskUpdAt),
				},
			}
			if stateID != nil {
				task.State = &models.StateName{ID: *stateID, Name: derefStr(stateName), Color: stateColor}
			}
			columnsMap[*colID].Tasks = append(columnsMap[*colID].Tasks, task)
		}
	}

	err = rows.Err()

	if err != nil {
		return models.WorkspaceDetail{}, err
	}

	if !initialized {
		return models.WorkspaceDetail{}, pgx.ErrNoRows
	}

	detail.Columns = make([]models.ColumnWithTasks, 0, len(columnOrder))
	for _, colID := range columnOrder {
		detail.Columns = append(detail.Columns, *columnsMap[colID])
	}

	stateRows, err := r.db.Query(ctx, `
		SELECT id, 
		       name, 
		       color
		FROM states
		WHERE user_id = $1
		ORDER BY name
		`,
		userID)
	if err != nil {
		return models.WorkspaceDetail{}, err
	}
	detail.States, err = pgx.CollectRows(stateRows, pgx.RowToStructByName[models.StateName])
	if err != nil {
		return models.WorkspaceDetail{}, err
	}

	return detail, nil
}

func (r *WorkspaceRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, name *string, description *string) (models.Workspace, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE workspaces
		SET name        = COALESCE($1, name),
		    description = COALESCE($2, description),
		    updated_at  = NOW()
		WHERE id = $3
		  AND user_id = $4
		  AND deleted_at IS NULL
		RETURNING id, name, description, user_id, created_at, updated_at, deleted_at
		`,
		name,
		description,
		id,
		userID)
	if err != nil {
		return models.Workspace{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Workspace])
}

func (r *WorkspaceRepository) Exists(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM workspaces
			WHERE id = $1 
			  AND user_id = $2 
			  AND deleted_at IS NULL
		)
		`,
		id,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

func (r *WorkspaceRepository) SoftDelete(ctx context.Context, id uuid.UUID, userID uuid.UUID) (models.Workspace, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Workspace{}, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		UPDATE workspaces
		SET deleted_at = NOW()
		WHERE id = $1
		  AND user_id = $2
		  AND deleted_at IS NULL
		RETURNING id, name, description, user_id, created_at, updated_at, deleted_at
		`,
		id,
		userID)
	if err != nil {
		return models.Workspace{}, err
	}
	workspace, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Workspace])
	if err != nil {
		return models.Workspace{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE columns
		SET deleted_at = NOW()
		WHERE workspace_id = $1 
		  AND deleted_at IS NULL
		`,
		id)

	if err != nil {
		return models.Workspace{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE tasks
		SET deleted_at = NOW()
		WHERE deleted_at IS NULL 
		  AND column_id 
		          IN (SELECT id FROM columns WHERE workspace_id = $1)
		`,
		id)

	if err != nil {
		return models.Workspace{}, err
	}

	err = tx.Commit(ctx)

	if err != nil {
		return models.Workspace{}, err
	}

	return workspace, nil
}

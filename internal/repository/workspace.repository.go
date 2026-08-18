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
	rows, err := r.db.Query(ctx,
		"SELECT id, title, description, user_id, created_at, updated_at FROM workspaces WHERE user_id = $1 ORDER BY created_at DESC",
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Workspace])
}

func (r *WorkspaceRepository) ListRecent(ctx context.Context, userID uuid.UUID) ([]models.Workspace, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, title, description, user_id, created_at, updated_at FROM workspaces WHERE user_id = $1 ORDER BY created_at DESC LIMIT 6",
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Workspace])
}

func (r *WorkspaceRepository) ListNames(ctx context.Context, userID uuid.UUID) ([]models.WorkspaceName, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, title FROM workspaces WHERE user_id = $1 ORDER BY created_at DESC",
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.WorkspaceName])
}

func (r *WorkspaceRepository) GetIDByTitle(ctx context.Context, title string, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		"SELECT id FROM workspaces WHERE title = $1 AND user_id = $2 ORDER BY created_at DESC LIMIT 1",
		title, userID).Scan(&id)
	return id, err
}

func (r *WorkspaceRepository) Create(ctx context.Context, title string, description *string, userID uuid.UUID) (models.Workspace, error) {
	rows, err := r.db.Query(ctx,
		"INSERT INTO workspaces (title, description, user_id) VALUES ($1, $2, $3) RETURNING *",
		title, description, userID)
	if err != nil {
		return models.Workspace{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Workspace])
}

func (r *WorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (models.WorkspaceDetail, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			w.id, w.title, w.description, w.user_id, w.created_at, w.updated_at,
			c.id, c.title, c.position, c.workspace_id, c.created_at, c.updated_at,
			t.id, t.title, t.description, t.position, t.column_id, t.created_at, t.updated_at
		FROM workspaces w
		LEFT JOIN columns c ON c.workspace_id = w.id
		LEFT JOIN tasks t ON t.column_id = c.id
		WHERE w.id = $1 AND w.user_id = $2
		ORDER BY c.position ASC, t.position ASC
	`, id, userID)
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
			wsTitle                    string
			wsDesc                     *string
			wsCreatedAt                time.Time
			wsUpdatedAt                *time.Time
			colID, colWsID             *uuid.UUID
			colTitle                   *string
			colPos                     *int
			colCreatedAt, colUpdatedAt *time.Time
			taskID, taskColID          *uuid.UUID
			taskTitle                  *string
			taskDesc                   *string
			taskPos                    *int
			taskCreatedAt, taskUpdAt   *time.Time
		)

		if err := rows.Scan(
			&wsID, &wsTitle, &wsDesc, &wsUserID, &wsCreatedAt, &wsUpdatedAt,
			&colID, &colTitle, &colPos, &colWsID, &colCreatedAt, &colUpdatedAt,
			&taskID, &taskTitle, &taskDesc, &taskPos, &taskColID, &taskCreatedAt, &taskUpdAt,
		); err != nil {
			return models.WorkspaceDetail{}, err
		}

		if !initialized {
			detail.Workspace = models.Workspace{
				ID:          wsID,
				Title:       wsTitle,
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
					Title:       derefStr(colTitle),
					Position:    derefInt(colPos),
					WorkspaceID: derefUUID(colWsID),
					CreatedAt:   derefTime(colCreatedAt),
					UpdatedAt:   derefTime(colUpdatedAt),
				},
				Tasks: []models.Task{},
			}
			columnsMap[*colID] = &col
			columnOrder = append(columnOrder, *colID)
		}

		if taskID != nil {
			task := models.Task{
				ID:          *taskID,
				Title:       derefStr(taskTitle),
				Description: taskDesc,
				Position:    derefInt(taskPos),
				ColumnID:    derefUUID(taskColID),
				CreatedAt:   derefTime(taskCreatedAt),
				UpdatedAt:   derefTime(taskUpdAt),
			}
			columnsMap[*colID].Tasks = append(columnsMap[*colID].Tasks, task)
		}
	}

	if err := rows.Err(); err != nil {
		return models.WorkspaceDetail{}, err
	}
	if !initialized {
		return models.WorkspaceDetail{}, pgx.ErrNoRows
	}

	detail.Columns = make([]models.ColumnWithTasks, 0, len(columnOrder))
	for _, colID := range columnOrder {
		detail.Columns = append(detail.Columns, *columnsMap[colID])
	}

	return detail, nil
}

func (r *WorkspaceRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, title *string, description *string) (models.Workspace, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE workspaces
		SET title       = COALESCE($1, title),
		    description = COALESCE($2, description),
		    updated_at  = NOW()
		WHERE id = $3 AND user_id = $4
		RETURNING *
	`, title, description, id, userID)
	if err != nil {
		return models.Workspace{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Workspace])
}

func (r *WorkspaceRepository) Exists(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = $1 AND user_id = $2)",
		id, userID).Scan(&exists)
	return exists, err
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func derefUUID(u *uuid.UUID) uuid.UUID {
	if u == nil {
		return uuid.UUID{}
	}
	return *u
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

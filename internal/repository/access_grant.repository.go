package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccessGrantRepository struct {
	db *pgxpool.Pool
}

func NewAccessGrantRepository(db *pgxpool.Pool) *AccessGrantRepository {
	return &AccessGrantRepository{db: db}
}

func (r *AccessGrantRepository) Grant(ctx context.Context, workspaceID uuid.UUID, columnID *uuid.UUID, memberID, grantedBy uuid.UUID, role string) (models.AccessGrant, error) {
	if columnID != nil {
		return queryStruct[models.AccessGrant](ctx, r.db, `
			INSERT INTO access_grants (workspace_id, column_id, member_id, role, granted_by)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (member_id, column_id) WHERE column_id IS NOT NULL
			DO UPDATE SET role = EXCLUDED.role
			RETURNING id, workspace_id, column_id, member_id, role, granted_by, created_at
			`,
			workspaceID,
			*columnID,
			memberID,
			role,
			grantedBy)
	}

	return queryStruct[models.AccessGrant](ctx, r.db, `
		INSERT INTO access_grants (workspace_id, column_id, member_id, role, granted_by)
		VALUES ($1, NULL, $2, $3, $4)
		ON CONFLICT (member_id, workspace_id) WHERE column_id IS NULL
		DO UPDATE SET role = EXCLUDED.role
		RETURNING id, workspace_id, column_id, member_id, role, granted_by, created_at
		`,
		workspaceID,
		memberID,
		role,
		grantedBy)
}

func (r *AccessGrantRepository) ListForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.AccessGrant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, workspace_id, column_id, member_id, role, granted_by, created_at
		FROM access_grants
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		`,
		workspaceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.AccessGrant])
}

func (r *AccessGrantRepository) HasAccess(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM access_grants
			WHERE workspace_id = $1
			  AND member_id = $2
		)
		`,
		workspaceID,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

func (r *AccessGrantRepository) HasWorkspaceAccess(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspaces w WHERE w.id = $1 AND w.created_by = $2
		) OR EXISTS(
			SELECT 1 FROM organisation_members om
			JOIN workspaces w ON w.organisation_id = om.organisation_id
			WHERE w.id = $1 AND om.member_id = $2
		) OR EXISTS(
			SELECT 1 FROM access_grants ag
			WHERE ag.workspace_id = $1 AND ag.column_id IS NULL AND ag.member_id = $2
		)
		`,
		workspaceID,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

func (r *AccessGrantRepository) HasColumnAccess(ctx context.Context, workspaceID, columnID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspaces w WHERE w.id = $1 AND w.created_by = $3
		) OR EXISTS(
			SELECT 1 FROM organisation_members om
			JOIN workspaces w ON w.organisation_id = om.organisation_id
			WHERE w.id = $1 AND om.member_id = $3
		) OR EXISTS(
			SELECT 1 FROM access_grants ag
			WHERE ag.workspace_id = $1 AND ag.column_id IS NULL AND ag.member_id = $3
		) OR EXISTS(
			SELECT 1 FROM access_grants ag
			WHERE ag.column_id = $2 AND ag.member_id = $3
		)
		`,
		workspaceID,
		columnID,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

func (r *AccessGrantRepository) HasWorkspaceEditAccess(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspaces w WHERE w.id = $1 AND w.created_by = $2
		) OR EXISTS(
			SELECT 1 FROM organisation_members om
			JOIN workspaces w ON w.organisation_id = om.organisation_id
			WHERE w.id = $1 AND om.member_id = $2 AND om.role = 'edit'
		) OR EXISTS(
			SELECT 1 FROM access_grants ag
			WHERE ag.workspace_id = $1 AND ag.column_id IS NULL AND ag.member_id = $2 AND ag.role = 'edit'
		)
		`,
		workspaceID,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

func (r *AccessGrantRepository) HasColumnEditAccess(ctx context.Context, workspaceID, columnID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM workspaces w WHERE w.id = $1 AND w.created_by = $3
		) OR EXISTS(
			SELECT 1 FROM organisation_members om
			JOIN workspaces w ON w.organisation_id = om.organisation_id
			WHERE w.id = $1 AND om.member_id = $3 AND om.role = 'edit'
		) OR EXISTS(
			SELECT 1 FROM access_grants ag
			WHERE ag.workspace_id = $1 AND ag.column_id IS NULL AND ag.member_id = $3 AND ag.role = 'edit'
		) OR EXISTS(
			SELECT 1 FROM access_grants ag
			WHERE ag.column_id = $2 AND ag.member_id = $3 AND ag.role = 'edit'
		)
		`,
		workspaceID,
		columnID,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

func (r *AccessGrantRepository) Revoke(ctx context.Context, grantID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM access_grants
		WHERE id = $1
		`,
		grantID)
	return err
}

package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrganisationRepository struct {
	db *pgxpool.Pool
}

func NewOrganisationRepository(db *pgxpool.Pool) *OrganisationRepository {
	return &OrganisationRepository{db: db}
}

func (r *OrganisationRepository) GetByUserWithWorkspaces(ctx context.Context, userID uuid.UUID) (models.Organisation, error) {
	var org models.Organisation

	row := r.db.QueryRow(ctx, `
		SELECT id, user_id
		FROM organisations
		WHERE user_id = $1
	`, userID)

	err := row.Scan(&org.ID, &org.UserID)
	if err != nil {
		return models.Organisation{}, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			name
		FROM workspaces
		WHERE user_id = $1
			AND deleted_at IS NULL
		ORDER BY COALESCE(updated_at, created_at) DESC
	`, userID)
	if err != nil {
		return models.Organisation{}, err
	}

	org.Workspaces, err = pgx.CollectRows(rows, pgx.RowToStructByName[models.WorkspaceName])
	if err != nil {
		return models.Organisation{}, err
	}
	if org.Workspaces == nil {
		org.Workspaces = []models.WorkspaceName{}
	}

	return org, nil
}

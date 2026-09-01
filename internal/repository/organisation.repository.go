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
			om.joined_at
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

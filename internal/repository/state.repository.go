package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StateRepository struct {
	db *pgxpool.Pool
}

func NewStateRepository(db *pgxpool.Pool) *StateRepository {
	return &StateRepository{db: db}
}

func (r *StateRepository) List(ctx context.Context, userID uuid.UUID) ([]models.State, error) {
	rows, err := r.db.Query(ctx, "SELECT * FROM states WHERE user_id = $1 ORDER BY name", userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.State])
}

func (r *StateRepository) Create(ctx context.Context, name string, color *string, userID uuid.UUID) (models.State, error) {
	rows, err := r.db.Query(ctx, "INSERT INTO states (name, color, user_id) VALUES ($1, $2, $3) RETURNING *",
		name, color, userID)
	if err != nil {
		return models.State{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.State])
}

func (r *StateRepository) GetOrCreate(ctx context.Context, userID uuid.UUID, name string) (models.State, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO states (name, user_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, name) DO UPDATE SET name = EXCLUDED.name
		RETURNING *
	`, name, userID)
	if err != nil {
		return models.State{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.State])
}

func (r *StateRepository) Exists(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM states WHERE id = $1 AND user_id = $2)", id, userID)
	err := row.Scan(&exists)
	return exists, err
}

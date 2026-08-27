package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, email, name, created_at, avatar_version, avatar_updated_at
		FROM users
		WHERE id = $1
		`,
		id)
	if err != nil {
		return models.User{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
}

func (r *UserRepository) UpdateName(ctx context.Context, id uuid.UUID, name string) (models.User, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE users
		SET name = $1
		WHERE id = $2
		RETURNING id, email, name, created_at, avatar_version, avatar_updated_at
		`,
		name,
		id)
	if err != nil {
		return models.User{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
}

func (r *UserRepository) SetAvatar(ctx context.Context, id uuid.UUID, version string) (models.User, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE users
		SET avatar_version    = $1,
		    avatar_updated_at = NOW()
		WHERE id = $2
		RETURNING id, email, name, created_at, avatar_version, avatar_updated_at
		`,
		version,
		id)
	if err != nil {
		return models.User{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
}

func (r *UserRepository) ClearAvatar(ctx context.Context, id uuid.UUID) (models.User, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE users
		SET avatar_version    = NULL,
		    avatar_updated_at  = NULL
		WHERE id = $1
		RETURNING id, email, name, created_at, avatar_version, avatar_updated_at
		`,
		id)
	if err != nil {
		return models.User{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
}

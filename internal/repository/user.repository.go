package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return queryStruct[models.User](ctx, r.db, `
		SELECT id, email, name, created_at, avatar_version, avatar_updated_at
		FROM users
		WHERE id = $1
		`,
		id)
}

func (r *UserRepository) UpdateName(ctx context.Context, id uuid.UUID, name string) (models.User, error) {
	return queryStruct[models.User](ctx, r.db, `
		UPDATE users
		SET name = $1
		WHERE id = $2
		RETURNING id, email, name, created_at, avatar_version, avatar_updated_at
		`,
		name,
		id)
}

func (r *UserRepository) SetAvatar(ctx context.Context, id uuid.UUID, version string) (models.User, error) {
	return queryStruct[models.User](ctx, r.db, `
		UPDATE users
		SET avatar_version    = $1,
		    avatar_updated_at = NOW()
		WHERE id = $2
		RETURNING id, email, name, created_at, avatar_version, avatar_updated_at
		`,
		version,
		id)
}

func (r *UserRepository) ClearAvatar(ctx context.Context, id uuid.UUID) (models.User, error) {
	return queryStruct[models.User](ctx, r.db, `
		UPDATE users
		SET avatar_version    = NULL,
		    avatar_updated_at  = NULL
		WHERE id = $1
		RETURNING id, email, name, created_at, avatar_version, avatar_updated_at
		`,
		id)
}

package repository

import (
	"context"
	"kanbano-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TagRepository struct {
	db *pgxpool.Pool
}

func NewTagRepository(db *pgxpool.Pool) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) List(ctx context.Context, userID uuid.UUID) ([]models.Tag, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, color, created_at, updated_at, user_id
		FROM tags
		WHERE user_id = $1
		ORDER BY name
		`,
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Tag])
}

func (r *TagRepository) Create(ctx context.Context, name string, color *string, userID uuid.UUID) (models.Tag, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO tags (name, color, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, color, user_id, created_at, updated_at
		`,
		name,
		color,
		userID)
	if err != nil {
		return models.Tag{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Tag])
}

func (r *TagRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, name *string, color *string) (models.Tag, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE tags
		SET name       = COALESCE($1, name),
		    color      = COALESCE($2, color),
		    updated_at = NOW()
		WHERE id = $3 AND user_id = $4
		RETURNING id, name, color, user_id, created_at, updated_at
		`,
		name,
		color,
		id,
		userID)
	if err != nil {
		return models.Tag{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Tag])
}

func (r *TagRepository) GetOrCreate(ctx context.Context, userID uuid.UUID, name string) (models.Tag, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO tags (name, user_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, name)
		    DO UPDATE SET name = EXCLUDED.name
		RETURNING id, name, color, user_id, created_at, updated_at
		`,
		name,
		userID)
	if err != nil {
		return models.Tag{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Tag])
}

func (r *TagRepository) GetByID(ctx context.Context, id uuid.UUID) (models.TagName, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id,
		       name,
		       color
		FROM tags
		WHERE id = $1
		`,
		id)
	if err != nil {
		return models.TagName{}, err
	}
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[models.TagName])
}

func (r *TagRepository) Exists(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM tags
			WHERE id = $1
			  AND user_id = $2
		)
		`,
		id,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

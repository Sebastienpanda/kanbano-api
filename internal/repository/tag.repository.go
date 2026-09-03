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
		SELECT id, name, color, created_by, updated_by, deleted_by, created_at, updated_at
		FROM tags
		WHERE created_by = $1
		ORDER BY name
		`,
		userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Tag])
}

func (r *TagRepository) Create(ctx context.Context, name string, color *string, userID uuid.UUID) (models.Tag, error) {
	return queryStruct[models.Tag](ctx, r.db, `
		INSERT INTO tags (name, color, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, name, color, created_by, updated_by, deleted_by, created_at, updated_at
		`,
		name,
		color,
		userID)
}

func (r *TagRepository) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, name *string, color *string) (models.Tag, error) {
	return queryStruct[models.Tag](ctx, r.db, `
		UPDATE tags
		SET name       = COALESCE($1, name),
		    color      = COALESCE($2, color),
		    updated_by = $4,
		    updated_at = NOW()
		WHERE id = $3 AND created_by = $4
		RETURNING id, name, color, created_by, updated_by, deleted_by, created_at, updated_at
		`,
		name,
		color,
		id,
		userID)
}

func (r *TagRepository) GetOrCreate(ctx context.Context, userID uuid.UUID, name string) (models.Tag, error) {
	return queryStruct[models.Tag](ctx, r.db, `
		INSERT INTO tags (name, created_by)
		VALUES ($1, $2)
		ON CONFLICT (created_by, name)
		    DO UPDATE SET name = EXCLUDED.name
		RETURNING id, name, color, created_by, updated_by, deleted_by, created_at, updated_at
		`,
		name,
		userID)
}

func (r *TagRepository) GetByID(ctx context.Context, id uuid.UUID) (models.TagName, error) {
	return queryStruct[models.TagName](ctx, r.db, `
		SELECT id,
		       name,
		       color
		FROM tags
		WHERE id = $1
		`,
		id)
}

func (r *TagRepository) Exists(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	var exists bool
	row := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM tags
			WHERE id = $1
			  AND created_by = $2
		)
		`,
		id,
		userID)
	err := row.Scan(&exists)
	return exists, err
}

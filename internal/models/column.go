package models

import (
	"time"

	"github.com/google/uuid"
)

type ColumnName struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
}

type Column struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Position    int        `json:"position"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	UpdatedBy   *uuid.UUID `json:"updated_by" extensions:"x-nullable"`
	DeletedBy   *uuid.UUID `json:"deleted_by" extensions:"x-nullable"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at" extensions:"x-nullable"`
	DeletedAt   *time.Time `json:"deleted_at" extensions:"x-nullable"`
}

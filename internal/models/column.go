package models

import (
	"time"

	"github.com/google/uuid"
)

type ColumnName struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Title       string    `json:"title"`
}

type Column struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Position    int       `json:"position"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

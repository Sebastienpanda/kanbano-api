package models

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Position    int        `json:"position"`
	ColumnID    uuid.UUID  `json:"column_id"`
	TagID       *uuid.UUID `json:"tag_id"`
	Status      *string    `json:"status"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	UpdatedBy   *uuid.UUID `json:"updated_by"`
	DeletedBy   *uuid.UUID `json:"deleted_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

package models

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID                uuid.UUID  `json:"id"`
	Name              string     `json:"name"`
	Description       *string    `json:"description"`
	Position          int        `json:"position"`
	ColumnID          uuid.UUID  `json:"column_id"`
	StateID           *uuid.UUID `json:"state_id"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	PositionUpdatedAt *time.Time `json:"position_updated_at"`
	DeletedAt         *time.Time `json:"deleted_at"`
}

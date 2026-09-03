package models

import (
	"time"

	"github.com/google/uuid"
)

type TagName struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Color *string   `json:"color"`
}

type Tag struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Color     *string    `json:"color"`
	CreatedBy uuid.UUID  `json:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by"`
	DeletedBy *uuid.UUID `json:"deleted_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

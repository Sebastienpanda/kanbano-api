package models

import (
	"time"

	"github.com/google/uuid"
)

type TagName struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Color *string   `json:"color" extensions:"x-nullable"`
}

type Tag struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Color     *string    `json:"color" extensions:"x-nullable"`
	CreatedBy uuid.UUID  `json:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by" extensions:"x-nullable"`
	DeletedBy *uuid.UUID `json:"deleted_by" extensions:"x-nullable"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at" extensions:"x-nullable"`
}

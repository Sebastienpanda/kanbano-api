package models

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description" extensions:"x-nullable"`
	Position    int        `json:"position"`
	ColumnID    uuid.UUID  `json:"column_id"`
	TagID       *uuid.UUID `json:"tag_id" extensions:"x-nullable"`
	Status      *string    `json:"status" enums:"À faire,En cours,Terminé" extensions:"x-nullable"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	UpdatedBy   *uuid.UUID `json:"updated_by" extensions:"x-nullable"`
	DeletedBy   *uuid.UUID `json:"deleted_by" extensions:"x-nullable"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at" extensions:"x-nullable"`
	DeletedAt   *time.Time `json:"deleted_at" extensions:"x-nullable"`
}

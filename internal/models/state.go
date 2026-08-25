package models

import (
	"time"

	"github.com/google/uuid"
)

type StateName struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Color *string   `json:"color"`
}

type State struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

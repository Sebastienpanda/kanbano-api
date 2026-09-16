package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	Name            *string    `json:"name"`
	CreatedAt       time.Time  `json:"created_at"`
	AvatarVersion   *string    `json:"-"`
	AvatarUpdatedAt *time.Time `json:"avatar_updated_at,omitempty"`
}

type AvatarSet struct {
	Size45  AvatarFormats `json:"45"`
	Size100 AvatarFormats `json:"100"`
}

type AvatarFormats struct {
	Avif string `json:"avif"`
	Webp string `json:"webp"`
	Png  string `json:"png"`
}

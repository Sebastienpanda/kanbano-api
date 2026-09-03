package models

import (
	"time"

	"github.com/google/uuid"
)

type Organisation struct {
	ID      uuid.UUID            `json:"id"`
	UserID  uuid.UUID            `json:"user_id"`
	Members []OrganisationMember `json:"members"`
}

// OrganisationMember rassemble les infos d'un utilisateur membre d'une
// organisation. AvatarVersion sert au handler pour construire les URLs d'avatar,
// il n'est pas exposé tel quel.
type OrganisationMember struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Name          *string    `json:"name" db:"name"`
	AvatarVersion *string    `json:"-" db:"avatar_version"`
	JoinedAt      *time.Time `json:"joined_at" db:"joined_at"`
}

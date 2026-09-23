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

type OrganisationMember struct {
	ID            uuid.UUID  `json:"id"`
	Name          *string    `json:"name"`
	AvatarVersion *string    `json:"-"`
	JoinedAt      *time.Time `json:"joined_at"`
	Role          string     `json:"role"`
}

// MemberProfile is a member's identity plus their role in the organisation
// and, per workspace of that organisation, their effective role there.
type MemberProfile struct {
	ID            uuid.UUID       `json:"id"`
	Email         string          `json:"email"`
	Name          *string         `json:"name"`
	AvatarVersion *string         `json:"-"`
	Role          string          `json:"role"`
	Workspaces    []WorkspaceRole `json:"workspaces"`
}

type WorkspaceRole struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	Role       string    `json:"role"`
	Visibility string    `json:"visibility"`
}

type OrganisationInvitation struct {
	ID             uuid.UUID  `json:"id"`
	OrganisationID uuid.UUID  `json:"organisation_id"`
	Email          string     `json:"email"`
	InvitedBy      uuid.UUID  `json:"invited_by"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	RespondedAt    *time.Time `json:"responded_at"`
	Role           *string    `json:"role"`
}

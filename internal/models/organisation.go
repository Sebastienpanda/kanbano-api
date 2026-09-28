package models

import (
	"time"

	"github.com/google/uuid"
)

// OrganisationSummary is an organisation the caller belongs to, with their
// role in it.
type OrganisationSummary struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role" enums:"owner,admin,member"`
}

type Organisation struct {
	ID      uuid.UUID            `json:"id"`
	Name    string               `json:"name"`
	UserID  uuid.UUID            `json:"user_id"`
	Members []OrganisationMember `json:"members"`
}

// OrganisationMember is a person of the organisation. The owner comes
// first, with the 'owner' role and no joined_at.
type OrganisationMember struct {
	ID            uuid.UUID  `json:"id"`
	Name          *string    `json:"name" extensions:"x-nullable"`
	AvatarVersion *string    `json:"-"`
	JoinedAt      *time.Time `json:"joined_at" extensions:"x-nullable"`
	Role          string     `json:"role" enums:"owner,admin,member"`
}

// MemberProfile is a member's identity plus their role in the organisation
// and, per workspace of that organisation, their effective role there.
type MemberProfile struct {
	ID            uuid.UUID       `json:"id"`
	Email         string          `json:"email"`
	Name          *string         `json:"name" extensions:"x-nullable"`
	AvatarVersion *string         `json:"-"`
	Role          string          `json:"role" enums:"owner,admin,member"`
	Workspaces    []WorkspaceRole `json:"workspaces"`
}

type WorkspaceRole struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	Role       string    `json:"role" enums:"view,edit"`
	Visibility string    `json:"visibility" enums:"private,public"`
}

type OrganisationInvitation struct {
	ID             uuid.UUID  `json:"id"`
	OrganisationID uuid.UUID  `json:"organisation_id"`
	Email          string     `json:"email"`
	InvitedBy      uuid.UUID  `json:"invited_by"`
	Status         string     `json:"status" enums:"pending,accepted,declined"`
	CreatedAt      time.Time  `json:"created_at"`
	RespondedAt    *time.Time `json:"responded_at" extensions:"x-nullable"`
	Role           *string    `json:"role" enums:"admin,member" extensions:"x-nullable"`
}

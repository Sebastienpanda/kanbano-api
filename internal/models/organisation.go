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

type OrganisationInvitation struct {
	ID             uuid.UUID  `json:"id"`
	OrganisationID uuid.UUID  `json:"organisation_id"`
	Email          string     `json:"email"`
	InvitedBy      uuid.UUID  `json:"invited_by"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	RespondedAt    *time.Time `json:"responded_at"`
	WorkspaceID    *uuid.UUID `json:"workspace_id"`
	ColumnID       *uuid.UUID `json:"column_id"`
	TaskID         *uuid.UUID `json:"task_id"`
	Role           *string    `json:"role"`
}

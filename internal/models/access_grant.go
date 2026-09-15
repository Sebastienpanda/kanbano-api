package models

import (
	"time"

	"github.com/google/uuid"
)

type AccessGrant struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	ColumnID    *uuid.UUID `json:"column_id"`
	MemberID    uuid.UUID  `json:"member_id"`
	Role        string     `json:"role"`
	GrantedBy   uuid.UUID  `json:"granted_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

type TaskAssignee struct {
	ID            uuid.UUID `json:"id"`
	Name          *string   `json:"name"`
	Email         string    `json:"email"`
	AvatarVersion *string   `json:"-"`
	Role          string    `json:"role"`
}

type TaskAssignedUser struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	AvatarVersion *string    `json:"-"`
	Avatar        *AvatarSet `json:"avatar"`
}

package models

import "github.com/google/uuid"

type Organisation struct {
	ID         uuid.UUID       `json:"id"`
	Workspaces []WorkspaceName `json:"workspaces"`
	UserID     uuid.UUID       `json:"user_id"`
}

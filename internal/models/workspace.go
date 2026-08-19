package models

import (
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	UserID      uuid.UUID  `json:"user_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type WorkspaceName struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type WorkspaceDetail struct {
	Workspace
	Columns []ColumnWithTasks `json:"columns"`
}

type ColumnWithTasks struct {
	Column
	Tasks []Task `json:"tasks"`
}

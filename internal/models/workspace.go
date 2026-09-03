package models

import (
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Description    *string    `json:"description"`
	OrganisationID uuid.UUID  `json:"organisation_id"`
	CreatedBy      uuid.UUID  `json:"created_by"`
	UpdatedBy      *uuid.UUID `json:"updated_by"`
	DeletedBy      *uuid.UUID `json:"deleted_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at"`
}

type WorkspaceName struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type WorkspaceSearchResult struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
}

type WorkspaceDetail struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   *time.Time        `json:"updated_at"`
	Columns     []ColumnWithTasks `json:"columns"`
}

type ColumnWithTasks struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	Position  int           `json:"position"`
	CreatedBy uuid.UUID     `json:"created_by"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt *time.Time    `json:"updated_at"`
	Tasks     []TaskWithTag `json:"tasks"`
}

type TaskWithTag struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Position    int        `json:"position"`
	ColumnID    uuid.UUID  `json:"column_id"`
	TagID       *uuid.UUID `json:"tag_id"`
	Status      *string    `json:"status"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	Tag         *TagName   `json:"tag"`
}

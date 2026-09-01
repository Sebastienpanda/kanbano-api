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
	Workspace
	Columns []ColumnWithTasks `json:"columns"`
	Tags    []TagName         `json:"tags"`
}

type ColumnWithTasks struct {
	Column
	Tasks []TaskWithTag `json:"tasks"`
}

type TaskWithTag struct {
	Task
	Tag *TagName `json:"tag"`
}

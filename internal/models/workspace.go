package models

import (
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Description    *string    `json:"description" extensions:"x-nullable"`
	OrganisationID uuid.UUID  `json:"organisation_id"`
	CreatedBy      uuid.UUID  `json:"created_by"`
	UpdatedBy      *uuid.UUID `json:"updated_by" extensions:"x-nullable"`
	DeletedBy      *uuid.UUID `json:"deleted_by" extensions:"x-nullable"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at" extensions:"x-nullable"`
	DeletedAt      *time.Time `json:"deleted_at" extensions:"x-nullable"`
}

type WorkspaceName struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type WorkspaceSearchResult struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description" extensions:"x-nullable"`
}

type WorkspaceDetail struct {
	ID             uuid.UUID         `json:"id"`
	OrganisationID uuid.UUID         `json:"organisation_id"`
	Name           string            `json:"name"`
	Description    *string           `json:"description" extensions:"x-nullable"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      *time.Time        `json:"updated_at" extensions:"x-nullable"`
	Members        []WorkspaceMember `json:"members"`
	Columns        []ColumnWithTasks `json:"columns"`
}

// WorkspaceMember is an admin or member granted access to a workspace
// (workspace_members.visibility = 'public'). The owner, who sees every
// workspace of the organisation, is never listed.
type WorkspaceMember struct {
	ID               uuid.UUID  `json:"id"`
	Name             *string    `json:"name" extensions:"x-nullable"`
	Email            string     `json:"email"`
	AvatarVersion    *string    `json:"-"`
	Avatar           *AvatarSet `json:"avatar" extensions:"x-nullable"`
	Role             string     `json:"role" enums:"view,edit"`
	OrganisationRole string     `json:"organisation_role" enums:"admin,member"`
}

type ColumnWithTasks struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	Position  int           `json:"position"`
	CreatedBy uuid.UUID     `json:"created_by"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt *time.Time    `json:"updated_at" extensions:"x-nullable"`
	Tasks     []TaskWithTag `json:"tasks"`
}

type TaskWithTag struct {
	ID            uuid.UUID          `json:"id"`
	Name          string             `json:"name"`
	Description   *string            `json:"description" extensions:"x-nullable"`
	Position      int                `json:"position"`
	ColumnID      uuid.UUID          `json:"column_id"`
	TagID         *uuid.UUID         `json:"tag_id" extensions:"x-nullable"`
	Status        *string            `json:"status" enums:"À faire,En cours,Terminé" extensions:"x-nullable"`
	CreatedBy     uuid.UUID          `json:"created_by"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     *time.Time         `json:"updated_at" extensions:"x-nullable"`
	Tag           *TagName           `json:"tag" extensions:"x-nullable"`
	AssignedUsers []TaskAssignedUser `json:"assigned_users"`
}

package models

import (
	"github.com/google/uuid"
)

// TaskRole is a task's id, column and the resolved effective role
// ('edit'/'view') of the requesting member on it.
type TaskRole struct {
	TaskID   uuid.UUID `json:"task_id"`
	ColumnID uuid.UUID `json:"column_id"`
	Role     string    `json:"role" enums:"view,edit"`
}

type TaskAssignee struct {
	ID            uuid.UUID `json:"id"`
	Name          *string   `json:"name" extensions:"x-nullable"`
	Email         string    `json:"email"`
	AvatarVersion *string   `json:"-"`
	IsGuest       bool      `json:"is_guest"`
}

type TaskAssignedUser struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	AvatarVersion *string    `json:"-"`
	Avatar        *AvatarSet `json:"avatar" extensions:"x-nullable"`
}

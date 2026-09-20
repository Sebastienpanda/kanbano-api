package models

import (
	"time"

	"github.com/google/uuid"
)

// TaskGuest is an external, non-organisation-member invitation scoped to a
// single task. UserID stays nil until the invitation is accepted.
type TaskGuest struct {
	ID          uuid.UUID  `json:"id"`
	TaskID      uuid.UUID  `json:"task_id"`
	Email       string     `json:"email"`
	UserID      *uuid.UUID `json:"user_id"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	InvitedBy   uuid.UUID  `json:"invited_by"`
	CreatedAt   time.Time  `json:"created_at"`
	RespondedAt *time.Time `json:"responded_at"`
}

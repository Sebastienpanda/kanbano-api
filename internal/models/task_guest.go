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
	UserID      *uuid.UUID `json:"user_id" extensions:"x-nullable"`
	Role        string     `json:"role" enums:"view,edit"`
	Status      string     `json:"status" enums:"pending,accepted,declined"`
	InvitedBy   uuid.UUID  `json:"invited_by"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	RespondedAt *time.Time `json:"responded_at" extensions:"x-nullable"`
}

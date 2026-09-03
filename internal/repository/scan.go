package repository

import (
	"time"

	"github.com/google/uuid"
)

// Helpers de déréférencement utilisés pour le scan manuel des LEFT JOIN
// (colonnes nullable) dans WorkspaceRepository.GetByID.

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func derefUUID(u *uuid.UUID) uuid.UUID {
	if u == nil {
		return uuid.UUID{}
	}
	return *u
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

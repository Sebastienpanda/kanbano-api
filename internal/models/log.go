package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Log struct {
	ID        uuid.UUID       `json:"id"`
	Level     string          `json:"level"`
	Message   string          `json:"message"`
	Source    string          `json:"source"`
	UserID    *uuid.UUID      `json:"userId"`
	RequestID *uuid.UUID      `json:"requestId"`
	Metadata  json.RawMessage `json:"metadata" swaggertype:"object"`
	CreatedAt time.Time       `json:"createdAt"`
}

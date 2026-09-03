package ws

import (
	"sync"

	"github.com/google/uuid"
)

type EventType = string

const (
	WorkspaceCreated EventType = "workspace.created"
	WorkspaceUpdated EventType = "workspace.updated"
	WorkspaceDeleted EventType = "workspace.deleted"
	TaskCreated      EventType = "task.created"
	TaskUpdated      EventType = "task.updated"
	TaskDeleted      EventType = "task.deleted"
	ColumnCreated    EventType = "column.created"
	ColumnUpdated    EventType = "column.updated"
	ColumnDeleted    EventType = "column.deleted"
	UserUpdated      EventType = "user.updated"
	AvatarUpdated    EventType = "avatar.updated"
	AvatarDeleted    EventType = "avatar.deleted"
)

type Event struct {
	Type        EventType  `json:"type"`
	WorkspaceID *uuid.UUID `json:"workspace_id,omitempty"`
	Data        any        `json:"data,omitempty"`
	Recent      any        `json:"recent,omitempty"`
}

type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[uuid.UUID]map[*Client]struct{})}
}

func (h *Hub) register(userID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][c] = struct{}{}
}

func (h *Hub) unregister(userID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[userID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.clients, userID)
		}
	}
}

func (h *Hub) Broadcast(userID uuid.UUID, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- event:
		default:
		}
	}
}

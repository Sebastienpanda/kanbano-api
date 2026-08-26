package handler

import (
	"net/http"
	"strings"

	"kanbano-api/internal/middleware"
	"kanbano-api/internal/ws"

	"github.com/google/uuid"
)

type WSHandler struct {
	hub *ws.Hub
}

func NewWSHandler(hub *ws.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

func bearerToken(r *http.Request) string {
	parts := strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) != "bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func (h *WSHandler) Serve(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	userIDStr, err := middleware.ValidateToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	ws.ServeWS(h.hub, userID, w, r)
}

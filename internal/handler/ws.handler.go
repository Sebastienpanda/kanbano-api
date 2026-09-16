package handler

import (
	"kanbano-api/internal/middleware"
	"kanbano-api/internal/ws"
	"net/http"
	"strings"

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

// Serve godoc
// @Summary WebSocket connection for real-time updates
// @Description Upgrades to a WebSocket connection. The JWT is passed via the Sec-WebSocket-Protocol header as "bearer, <token>" (not a normal Authorization header, so no BearerAuth security scheme applies here).
// @Tags ws
// @Success 101 "Switching Protocols"
// @Failure 401 {object} string "missing or invalid token"
// @Router /ws [get]
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

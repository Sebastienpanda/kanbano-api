package routes

import (
	"kanbano-api/internal/handler"

	"github.com/go-chi/chi/v5"
)

func RoleRoutes(r chi.Router, h *handler.RoleHandler) {
	r.Get("/roles", h.Role)
}

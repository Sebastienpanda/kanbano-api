package routes

import (
	"kanbano-api/internal/handler"

	"github.com/go-chi/chi/v5"
)

func HealthRoutes(r chi.Router) {
	r.Get("/health", handler.Health)
}

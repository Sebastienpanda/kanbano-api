package routes

import (
	"kanbano-api/internal/handler"

	"github.com/go-chi/chi/v5"
)

func BrevoRoutes(r chi.Router, h *handler.BrevoHandler) {
	r.Post("/webhooks/brevo", h.Webhook)
}

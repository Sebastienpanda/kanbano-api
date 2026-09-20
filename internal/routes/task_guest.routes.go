package routes

import (
	"kanbano-api/internal/handler"

	"github.com/go-chi/chi/v5"
)

func TaskGuestRoutes(r chi.Router, h *handler.TaskGuestHandler) {
	r.Route("/guest-invitations", func(r chi.Router) {
		r.Get("/received", h.ReceivedInvitations)
		r.Patch("/{id}", h.UpdateInvitationStatus)
	})
}

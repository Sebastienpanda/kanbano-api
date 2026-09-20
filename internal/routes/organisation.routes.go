package routes

import (
	"kanbano-api/internal/handler"

	"github.com/go-chi/chi/v5"
)

func OrganisationRoutes(r chi.Router, oh *handler.OrganisationHandler) {
	r.Route("/organisation", func(r chi.Router) {
		r.Get("/", oh.Get)
		r.Get("/members/{id}/profile", oh.MemberProfile)

		r.Route("/invitations", func(r chi.Router) {
			r.Post("/", oh.Invite)
			r.Get("/sent", oh.SentInvitations)
			r.Get("/received", oh.ReceivedInvitations)
			r.Patch("/{id}", oh.UpdateInvitationStatus)
		})
	})
}

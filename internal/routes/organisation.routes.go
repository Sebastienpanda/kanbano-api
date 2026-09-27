package routes

import (
	"kanbano-api/internal/handler"

	"github.com/go-chi/chi/v5"
)

func OrganisationRoutes(r chi.Router, oh *handler.OrganisationHandler) {
	r.Route("/organisations", func(r chi.Router) {
		r.Get("/", oh.List)

		r.Route("/{orgId}", func(r chi.Router) {
			r.Get("/", oh.Get)
			r.Patch("/", oh.Update)
			r.Delete("/", oh.Delete)

			r.Post("/invitations", oh.Invite)
			r.Get("/invitations", oh.SentInvitations)

			r.Patch("/members/{memberId}", oh.SetMemberRole)
			r.Delete("/members/{memberId}", oh.RemoveMember)
			r.Get("/members/{memberId}/profile", oh.MemberProfile)
		})
	})

	r.Route("/organisation-invitations", func(r chi.Router) {
		r.Get("/received", oh.ReceivedInvitations)
		r.Patch("/{id}", oh.UpdateInvitationStatus)
	})
}

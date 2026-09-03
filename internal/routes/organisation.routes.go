package routes

import (
	"kanbano-api/internal/handler"

	"github.com/go-chi/chi/v5"
)

func OrganisationRoutes(r chi.Router, oh *handler.OrganisationHandler) {
	r.Route("/organisation", func(r chi.Router) {

		r.Get("/", oh.Get)
	})
}

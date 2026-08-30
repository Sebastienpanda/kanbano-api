package routes

import (
	"kanbano-api/internal/handler"
	"kanbano-api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func OrganisationRoutes(r chi.Router, oh *handler.OrganisationHandler) {
	r.Route("/organisation", func(r chi.Router) {
		r.Use(middleware.AuthRequired)

		r.Get("/", oh.Get)
	})
}

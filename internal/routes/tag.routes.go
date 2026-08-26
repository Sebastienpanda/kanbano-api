package routes

import (
	"kanbano-api/internal/handler"
	"kanbano-api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func TagsRoutes(r chi.Router, wh *handler.TagHandler) {

	r.Route("/tags", func(r chi.Router) {
		r.Use(middleware.AuthRequired)

		r.Get("/", wh.List)
		r.Post("/", wh.Create)
		r.Patch("/{id}", wh.Update)
	})
}

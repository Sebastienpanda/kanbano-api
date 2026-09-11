package routes

import (
	"kanbano-api/internal/handler"
	"kanbano-api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func LogRoutes(r chi.Router, lh *handler.LogHandler) {

	r.Route("/admin/logs", func(r chi.Router) {
		r.Use(middleware.AdminRequired)

		r.Get("/", lh.List)
	})
}

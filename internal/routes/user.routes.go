package routes

import (
	"kanbano-api/internal/handler"
	"kanbano-api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func UsersRoutes(r chi.Router, uh *handler.UserHandler) {
	r.Route("/me", func(r chi.Router) {
		r.Use(middleware.AuthRequired)

		r.Get("/", uh.Me)
		r.Patch("/", uh.UpdateMe)
		r.Put("/avatar", uh.UploadAvatar)
		r.Delete("/avatar", uh.DeleteAvatar)
	})
}

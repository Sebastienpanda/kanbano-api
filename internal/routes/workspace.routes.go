package routes

import (
	"kanbano-api/internal/handler"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	Workspace *handler.WorkspaceHandler
	Column    *handler.ColumnHandler
	Task      *handler.TaskHandler
}

func Workspaces(r chi.Router, h Handlers) {
	r.Route("/workspaces", func(r chi.Router) {

		r.Get("/", h.Workspace.List)
		r.Get("/recent", h.Workspace.Recent)
		r.Get("/search", h.Workspace.Search)
		r.Get("/names", h.Workspace.Names)
		r.Post("/", h.Workspace.Create)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.Workspace.Get)
			r.Patch("/", h.Workspace.Update)
			r.Delete("/", h.Workspace.Delete)

			columns(r, h)
		})
	})
}

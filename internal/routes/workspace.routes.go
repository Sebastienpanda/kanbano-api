package routes

import (
	"kanbano-api/internal/handler"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	Workspace    *handler.WorkspaceHandler
	Column       *handler.ColumnHandler
	Task         *handler.TaskHandler
	TaskGuest    *handler.TaskGuestHandler
	Organisation *handler.OrganisationHandler
	Role         *handler.RoleHandler
}

func Workspaces(r chi.Router, h Handlers) {
	r.Route("/workspaces", func(r chi.Router) {
		r.Get("/", h.Workspace.List)
		r.Post("/", h.Workspace.Create)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.Workspace.Get)
			r.Patch("/", h.Workspace.Update)
			r.Delete("/", h.Workspace.Delete)

			r.Get("/tasks/roles", h.Task.Roles)
			r.Get("/abilities", h.Role.Abilities)
			r.Put("/members/{memberID}/role", h.Role.SetWorkspaceRole)

			columns(r, h)
		})
	})
}

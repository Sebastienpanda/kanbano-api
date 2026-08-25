package routes

import (
	"kanbano-api/internal/handler"
	"kanbano-api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func WorkspacesRoutes(r chi.Router, wh *handler.WorkspaceHandler, ch *handler.ColumnHandler, th *handler.TaskHandler) {
	r.Route("/workspaces", func(r chi.Router) {
		r.Use(middleware.AuthRequired)

		r.Get("/", wh.List)
		r.Get("/recent", wh.Recent)
		r.Get("/names", wh.Names)
		r.Get("/columns/names", ch.NamesByWorkspaceName)
		r.Post("/", wh.Create)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", wh.Get)
			r.Patch("/", wh.Update)
			r.Delete("/", wh.Delete)

			r.Post("/columns", ch.Create)
			r.Patch("/columns/{columnId}", ch.Update)
			r.Patch("/columns/{columnId}/reorder", ch.Reorder)
			r.Delete("/columns/{columnId}", ch.Delete)

			r.Post("/columns/{columnId}/tasks", th.Create)
			r.Patch("/columns/{columnId}/tasks/{taskId}", th.Update)
			r.Patch("/columns/{columnId}/tasks/{taskId}/reorder", th.Reorder)
			r.Delete("/columns/{columnId}/tasks/{taskId}", th.Delete)
		})
	})
}

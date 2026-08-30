package routes

import "github.com/go-chi/chi/v5"

func columns(r chi.Router, h Handlers) {
	r.Get("/columns/names", h.Column.NamesByWorkspaceName)
	r.Post("/columns", h.Column.Create)
	r.Patch("/columns/{columnId}", h.Column.Update)
	r.Patch("/columns/{columnId}/reorder", h.Column.Reorder)
	r.Delete("/columns/{columnId}", h.Column.Delete)

	task(r, h)
}

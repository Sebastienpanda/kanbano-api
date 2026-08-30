package routes

import "github.com/go-chi/chi/v5"

func task(r chi.Router, h Handlers) {
	r.Post("/columns/{columnId}/tasks", h.Task.Create)
	r.Patch("/columns/{columnId}/tasks/{taskId}", h.Task.Update)
	r.Patch("/columns/{columnId}/tasks/{taskId}/reorder", h.Task.Reorder)
	r.Delete("/columns/{columnId}/tasks/{taskId}", h.Task.Delete)
}

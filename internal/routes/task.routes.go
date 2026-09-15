package routes

import "github.com/go-chi/chi/v5"

func task(r chi.Router, h Handlers) {
	r.Post("/columns/{columnId}/tasks", h.Task.Create)
	r.Patch("/columns/{columnId}/tasks/{taskId}", h.Task.Update)
	r.Delete("/columns/{columnId}/tasks/{taskId}", h.Task.Delete)
	r.Get("/columns/{columnId}/tasks/{taskId}/assignees", h.Task.Assignees)
	r.Post("/columns/{columnId}/tasks/{taskId}/assignees", h.Task.Assign)
	r.Delete("/columns/{columnId}/tasks/{taskId}/assignees/{memberId}", h.Task.Unassign)
}

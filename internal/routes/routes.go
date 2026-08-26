package routes

import (
	"kanbano-api/internal/handler"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r *chi.Mux, pool *pgxpool.Pool) {
	hub := ws.NewHub()

	workspaceRepo := repository.NewWorkspaceRepository(pool)
	columnRepo := repository.NewColumnRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)
	tagRepo := repository.NewTagRepository(pool)

	workspaceHandler := handler.NewWorkspaceHandler(workspaceRepo, hub)
	columnHandler := handler.NewColumnHandler(columnRepo, workspaceRepo, hub)
	taskHandler := handler.NewTaskHandler(taskRepo, workspaceRepo, columnRepo, tagRepo, hub)
	tagHandler := handler.NewTagHandler(tagRepo)
	wsHandler := handler.NewWSHandler(hub)

	r.Route("/api/v1", func(r chi.Router) {
		WorkspacesRoutes(r, workspaceHandler, columnHandler, taskHandler)
		TagsRoutes(r, tagHandler)

		r.Get("/ws", wsHandler.Serve)
	})
}

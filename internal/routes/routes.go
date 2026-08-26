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
	stateRepo := repository.NewStateRepository(pool)

	workspaceHandler := handler.NewWorkspaceHandler(workspaceRepo, hub)
	columnHandler := handler.NewColumnHandler(columnRepo, workspaceRepo, hub)
	taskHandler := handler.NewTaskHandler(taskRepo, workspaceRepo, columnRepo, stateRepo, hub)
	stateHandler := handler.NewStateHandler(stateRepo)
	wsHandler := handler.NewWSHandler(hub)

	r.Route("/api/v1", func(r chi.Router) {
		WorkspacesRoutes(r, workspaceHandler, columnHandler, taskHandler)
		StatesRoutes(r, stateHandler)

		r.Get("/ws", wsHandler.Serve)
	})
}

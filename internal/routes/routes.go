package routes

import (
	"kanbano-api/internal/handler"
	"kanbano-api/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r *chi.Mux, pool *pgxpool.Pool) {
	workspaceRepo := repository.NewWorkspaceRepository(pool)
	columnRepo := repository.NewColumnRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)

	workspaceHandler := handler.NewWorkspaceHandler(workspaceRepo)
	columnHandler := handler.NewColumnHandler(columnRepo, workspaceRepo)
	taskHandler := handler.NewTaskHandler(taskRepo, workspaceRepo, columnRepo)

	r.Route("/api/v1", func(r chi.Router) {
		WorkspacesRoutes(r, workspaceHandler, columnHandler, taskHandler)
	})
}

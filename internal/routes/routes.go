package routes

import (
	"kanbano-api/internal/handler"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/storage"
	"kanbano-api/internal/ws"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r *chi.Mux, pool *pgxpool.Pool, store *storage.Client) {
	hub := ws.NewHub()

	workspaceRepo := repository.NewWorkspaceRepository(pool)
	columnRepo := repository.NewColumnRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)
	tagRepo := repository.NewTagRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	organisationRepo := repository.NewOrganisationRepository(pool)

	workspaceHandler := handler.NewWorkspaceHandler(workspaceRepo, hub)
	columnHandler := handler.NewColumnHandler(columnRepo, workspaceRepo, hub)
	taskHandler := handler.NewTaskHandler(taskRepo, workspaceRepo, columnRepo, tagRepo, hub)
	tagHandler := handler.NewTagHandler(tagRepo)
	userHandler := handler.NewUserHandler(userRepo, store, hub)
	organisationHandler := handler.NewOrganisationHandler(organisationRepo)
	wsHandler := handler.NewWSHandler(hub)
	brevoHandler := handler.NewBrevoHandler(os.Getenv("BREVO_WEBHOOK_SECRET"), os.Getenv("DISCORD_WEBHOOK_URL"))

	r.Route("/api/v1", func(r chi.Router) {
		WorkspacesRoutes(r, workspaceHandler, columnHandler, taskHandler)
		TagsRoutes(r, tagHandler)
		UsersRoutes(r, userHandler)
		OrganisationRoutes(r, organisationHandler)
		BrevoRoutes(r, brevoHandler)

		r.Get("/ws", wsHandler.Serve)
	})
}

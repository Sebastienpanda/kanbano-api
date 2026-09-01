package routes

import (
	"kanbano-api/internal/handler"
	"kanbano-api/internal/middleware"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/storage"
	"kanbano-api/internal/ws"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repositories struct {
	workspace    *repository.WorkspaceRepository
	column       *repository.ColumnRepository
	task         *repository.TaskRepository
	tag          *repository.TagRepository
	user         *repository.UserRepository
	organisation *repository.OrganisationRepository
}

func newRepositories(pool *pgxpool.Pool) repositories {
	return repositories{
		workspace:    repository.NewWorkspaceRepository(pool),
		column:       repository.NewColumnRepository(pool),
		task:         repository.NewTaskRepository(pool),
		tag:          repository.NewTagRepository(pool),
		user:         repository.NewUserRepository(pool),
		organisation: repository.NewOrganisationRepository(pool),
	}
}

type handlers struct {
	workspace    *handler.WorkspaceHandler
	column       *handler.ColumnHandler
	task         *handler.TaskHandler
	tag          *handler.TagHandler
	user         *handler.UserHandler
	organisation *handler.OrganisationHandler
	ws           *handler.WSHandler
	brevo        *handler.BrevoHandler
}

func newHandlers(repos repositories, store *storage.Client, hub *ws.Hub) handlers {
	return handlers{
		workspace:    handler.NewWorkspaceHandler(repos.workspace, hub),
		column:       handler.NewColumnHandler(repos.column, repos.workspace, hub),
		task:         handler.NewTaskHandler(repos.task, repos.workspace, repos.column, repos.tag, hub),
		tag:          handler.NewTagHandler(repos.tag),
		user:         handler.NewUserHandler(repos.user, store, hub),
		organisation: handler.NewOrganisationHandler(repos.organisation),
		ws:           handler.NewWSHandler(hub),
		brevo:        handler.NewBrevoHandler(os.Getenv("BREVO_WEBHOOK_SECRET"), os.Getenv("DISCORD_WEBHOOK_URL")),
	}
}

func RegisterRoutes(r *chi.Mux, pool *pgxpool.Pool, store *storage.Client) {
	hub := ws.NewHub()
	repos := newRepositories(pool)
	h := newHandlers(repos, store, hub)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.AuthRequired)

		Workspaces(r, Handlers{
			Workspace: h.workspace,
			Column:    h.column,
			Task:      h.task,
		})
		TagsRoutes(r, h.tag)
		UsersRoutes(r, h.user)
		OrganisationRoutes(r, h.organisation)
	})

	// /ws s'authentifie lui-même via Sec-WebSocket-Protocol (pas de header Authorization possible côté navigateur)
	r.Get("/api/v1/ws", h.ws.Serve)

	BrevoRoutes(r, h.brevo)
}

package routes

import (
	"kanbano-api/internal/brevo"
	"kanbano-api/internal/handler"
	"kanbano-api/internal/middleware"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/storage"
	"kanbano-api/internal/ws"
	"os"
	"strconv"

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
	role         *repository.RoleRepository
	taskAssignee *repository.TaskAssigneeRepository
	taskGuest    *repository.TaskGuestRepository
	log          *repository.LogRepository
}

func newRepositories(pool *pgxpool.Pool) repositories {
	return repositories{
		workspace:    repository.NewWorkspaceRepository(pool),
		column:       repository.NewColumnRepository(pool),
		task:         repository.NewTaskRepository(pool),
		tag:          repository.NewTagRepository(pool),
		user:         repository.NewUserRepository(pool),
		organisation: repository.NewOrganisationRepository(pool),
		role:         repository.NewRoleRepository(pool),
		taskAssignee: repository.NewTaskAssigneeRepository(pool),
		taskGuest:    repository.NewTaskGuestRepository(pool),
		log:          repository.NewLogRepository(pool),
	}
}

type handlers struct {
	workspace    *handler.WorkspaceHandler
	column       *handler.ColumnHandler
	task         *handler.TaskHandler
	taskGuest    *handler.TaskGuestHandler
	tag          *handler.TagHandler
	user         *handler.UserHandler
	organisation *handler.OrganisationHandler
	role         *handler.RoleHandler
	ws           *handler.WSHandler
	brevo        *handler.BrevoHandler
	log          *handler.LogHandler
}

func newHandlers(repos repositories, store *storage.Client, hub *ws.Hub) handlers {
	invitationTplID, _ := strconv.Atoi(os.Getenv("BREVO_INVITATION_TEMPLATE_ID"))
	mailer := brevo.NewClient(os.Getenv("BREVO_API_KEY"))

	return handlers{
		workspace: handler.NewWorkspaceHandler(repos.workspace, store, hub),
		column:    handler.NewColumnHandler(repos.column, repos.workspace, repos.role, hub),
		task: handler.NewTaskHandler(handler.TaskHandlerDeps{
			Repo:          repos.task,
			WorkspaceRepo: repos.workspace,
			ColumnRepo:    repos.column,
			TagRepo:       repos.tag,
			Role:          repos.role,
			TaskAssignee:  repos.taskAssignee,
			Store:         store,
			Hub:           hub,
		}),
		taskGuest: handler.NewTaskGuestHandler(handler.TaskGuestHandlerConfig{
			Repo:            repos.taskGuest,
			WorkspaceRepo:   repos.workspace,
			ColumnRepo:      repos.column,
			TaskRepo:        repos.task,
			UserRepo:        repos.user,
			Role:            repos.role,
			Mailer:          mailer,
			InvitationTplID: invitationTplID,
			FrontendBaseURL: os.Getenv("FRONTEND_URL"),
		}),
		tag:  handler.NewTagHandler(repos.tag),
		user: handler.NewUserHandler(repos.user, store, hub),
		organisation: handler.NewOrganisationHandler(handler.OrganisationHandlerConfig{
			Repo:            repos.organisation,
			UserRepo:        repos.user,
			Store:           store,
			Mailer:          mailer,
			InvitationTplID: invitationTplID,
			FrontendBaseURL: os.Getenv("FRONTEND_URL"),
		}),
		role:  handler.NewRoleHandler(repos.role, repos.organisation, repos.workspace),
		ws:    handler.NewWSHandler(hub),
		brevo: handler.NewBrevoHandler(os.Getenv("BREVO_WEBHOOK_SECRET"), os.Getenv("DISCORD_WEBHOOK_URL")),
		log:   handler.NewLogHandler(repos.log),
	}
}

func RegisterRoutes(r *chi.Mux, pool *pgxpool.Pool, store *storage.Client) {
	hub := ws.NewHub()
	repos := newRepositories(pool)
	h := newHandlers(repos, store, hub)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ws", h.ws.Serve)
		SwaggerRoutes(r)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthRequired)

			Workspaces(r, Handlers{
				Workspace:    h.workspace,
				Column:       h.column,
				Task:         h.task,
				TaskGuest:    h.taskGuest,
				Organisation: h.organisation,
				Role:         h.role,
			})
			TagsRoutes(r, h.tag)
			UsersRoutes(r, h.user)
			OrganisationRoutes(r, h.organisation)
			RoleRoutes(r, h.role)
			TaskGuestRoutes(r, h.taskGuest)
			LogRoutes(r, h.log)
		})
	})

	HealthRoutes(r)
	BrevoRoutes(r, h.brevo)
}

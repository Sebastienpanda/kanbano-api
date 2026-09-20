package handler

import (
	"context"
	"errors"
	"kanbano-api/internal/brevo"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/models"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/utils"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// TaskGuestHandler manages invitations for external guests scoped to a
// single task — people with no organisation membership at all. It is
// deliberately separate from TaskHandler.Assign/Assignees, which manage
// task_assignees for existing organisation members.
type TaskGuestHandler struct {
	repo            *repository.TaskGuestRepository
	workspaceRepo   *repository.WorkspaceRepository
	columnRepo      *repository.ColumnRepository
	taskRepo        *repository.TaskRepository
	userRepo        *repository.UserRepository
	role            *repository.RoleRepository
	mailer          *brevo.Client
	invitationTplID int
	frontendBaseURL string
}

type TaskGuestHandlerConfig struct {
	Repo            *repository.TaskGuestRepository
	WorkspaceRepo   *repository.WorkspaceRepository
	ColumnRepo      *repository.ColumnRepository
	TaskRepo        *repository.TaskRepository
	UserRepo        *repository.UserRepository
	Role            *repository.RoleRepository
	Mailer          *brevo.Client
	InvitationTplID int
	FrontendBaseURL string
}

func NewTaskGuestHandler(cfg TaskGuestHandlerConfig) *TaskGuestHandler {
	return &TaskGuestHandler{
		repo:            cfg.Repo,
		workspaceRepo:   cfg.WorkspaceRepo,
		columnRepo:      cfg.ColumnRepo,
		taskRepo:        cfg.TaskRepo,
		userRepo:        cfg.UserRepo,
		role:            cfg.Role,
		mailer:          cfg.Mailer,
		invitationTplID: cfg.InvitationTplID,
		frontendBaseURL: cfg.FrontendBaseURL,
	}
}

type inviteTaskGuestBody struct {
	Email string  `json:"email" validate:"required,email"`
	Role  *string `json:"role,omitempty" validate:"omitempty,oneof=view edit"`
}

type updateTaskGuestInvitationBody struct {
	Status string `json:"status" validate:"required,oneof=accepted declined"`
}

// Invite godoc
// @Summary Invite an external guest to a task
// @Description Invites someone by email to a single task, without granting any organisation membership. They can accept even without an existing Kanbano account.
// @Tags task
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param body body inviteTaskGuestBody true "Invitation to create"
// @Success 201 {object} utils.CreateResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/guests [post]
func (h *TaskGuestHandler) Invite(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	taskExists, err := h.taskRepo.Exists(r.Context(), taskID, columnID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if !taskExists {
		notFound(w, r, "task not found")
		return
	}

	hasEditAccess, err := h.role.HasWorkspaceEditAccess(r.Context(), workspaceID, userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if !hasEditAccess {
		forbidden(w, r, "edit access required")
		return
	}

	body, ok := utils.DecodeAndValidate[inviteTaskGuestBody]("TaskGuestHandler.Invite", w, r)
	if !ok {
		return
	}

	guest, err := h.repo.CreateInvitation(r.Context(), repository.TaskGuestInvitationParams{
		TaskID:    taskID,
		Email:     body.Email,
		InvitedBy: userID,
		Role:      body.Role,
	})
	if err != nil {
		if errors.Is(err, repository.ErrTaskGuestInvitationAlreadyPending) {
			conflict(w, r, "an invitation is already pending for this email")
			return
		}
		serverError(w, r, err)
		return
	}

	inviter, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		logging.Logger.Error("failed to load inviter for task guest invitation email", slog.Any("error", err))
	} else {
		h.sendInvitationEmail(guest, inviter)
	}

	utils.RespondCreated(w, &guest.ID)
}

func (h *TaskGuestHandler) parseTaskContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	userID, workspaceID, ok := requireWorkspace(w, r, h.workspaceRepo)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	columnID, err := uuid.Parse(chi.URLParam(r, "columnId"))
	if err != nil {
		badRequest(w, r, "invalid columnId")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	colExists, err := h.columnRepo.Exists(r.Context(), columnID, workspaceID)
	if err != nil {
		serverError(w, r, err)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if !colExists {
		notFound(w, r, "column not found")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	return userID, workspaceID, columnID, true
}

func (h *TaskGuestHandler) sendInvitationEmail(guest models.TaskGuest, inviter models.User) {
	if h.mailer == nil || h.invitationTplID == 0 {
		return
	}

	inviterName := inviter.Email
	if inviter.Name != nil && *inviter.Name != "" {
		inviterName = *inviter.Name
	}

	params := map[string]any{
		"INVITER_NAME":  inviterName,
		"INVITER_EMAIL": inviter.Email,
		"ACCEPT_URL":    h.frontendBaseURL + "/guest-invitations/" + guest.ID.String() + "?action=accept",
		"DECLINE_URL":   h.frontendBaseURL + "/guest-invitations/" + guest.ID.String() + "?action=decline",
	}

	safeGo(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.mailer.SendTemplateEmail(ctx, guest.Email, h.invitationTplID, params); err != nil {
			logging.Logger.Error("failed to send task guest invitation email", slog.Any("error", err))
		}
	})
}

// SentInvitations godoc
// @Summary List pending guest invitations sent for a task
// @Tags task
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.TaskGuest
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/guests [get]
func (h *TaskGuestHandler) SentInvitations(w http.ResponseWriter, r *http.Request) {
	_, _, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	taskExists, err := h.taskRepo.Exists(r.Context(), taskID, columnID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if !taskExists {
		notFound(w, r, "task not found")
		return
	}

	limit, offset, ok := parsePagination(w, r)
	if !ok {
		return
	}

	invitations, err := h.repo.ListSentInvitations(r.Context(), taskID, limit, offset)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, invitations)
}

// ReceivedInvitations godoc
// @Summary List task guest invitations received by the current user
// @Tags task
// @Produce json
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.TaskGuest
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /guest-invitations/received [get]
func (h *TaskGuestHandler) ReceivedInvitations(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	limit, offset, ok := parsePagination(w, r)
	if !ok {
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if handleRepoError(w, r, err, "user not found") {
		return
	}

	invitations, err := h.repo.ListReceivedInvitations(r.Context(), user.Email, limit, offset)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, invitations)
}

// UpdateInvitationStatus godoc
// @Summary Accept or decline a task guest invitation
// @Tags task
// @Accept json
// @Produce json
// @Param id path string true "Invitation ID"
// @Param body body updateTaskGuestInvitationBody true "New status"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /guest-invitations/{id} [patch]
func (h *TaskGuestHandler) UpdateInvitationStatus(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	invitationID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[updateTaskGuestInvitationBody]("TaskGuestHandler.UpdateInvitationStatus", w, r)
	if !ok {
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if handleRepoError(w, r, err, "user not found") {
		return
	}

	if body.Status == "accepted" {
		_, err = h.repo.AcceptInvitation(r.Context(), invitationID, userID, user.Email)
	} else {
		_, err = h.repo.DeclineInvitation(r.Context(), invitationID, user.Email)
	}
	if handleRepoError(w, r, err, "invitation not found") {
		return
	}

	utils.RespondUpdated(w)
}

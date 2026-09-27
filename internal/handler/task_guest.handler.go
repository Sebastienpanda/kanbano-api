package handler

import (
	"context"
	"errors"
	"kanbano-api/internal/brevo"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/models"
	"kanbano-api/internal/permissions"
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

type setTaskGuestRoleBody struct {
	Role string `json:"role" validate:"required,oneof=view edit"`
}

// Invite godoc
// @Summary Invite an external guest to a task
// @Description Invites someone by email to a single task, without granting any organisation membership. They can accept even without an existing Kanbano account. The invitation expires after 7 days; inviting the same email again replaces an expired invitation. Allowed for the organisation owner, or an admin the workspace is shared with.
// @Tags task
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param body body inviteTaskGuestBody true "Invitation to create"
// @Success 201 {object} utils.CreateResponse
// @Failure 400 {object} BadRequestResponse "invalid id | invalid columnId | invalid taskId | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found | task not found"
// @Failure 409 {object} ErrorResponse "an invitation is already pending for this email | this user already has access to this task"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/guests [post]
func (h *TaskGuestHandler) Invite(w http.ResponseWriter, r *http.Request) {
	userID, taskID, ok := h.requireGuestManagement(w, r)
	if !ok {
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
	switch {
	case errors.Is(err, repository.ErrTaskGuestInvitationAlreadyPending):
		conflict(w, r, "an invitation is already pending for this email")
		return
	case errors.Is(err, repository.ErrTaskGuestAlreadyHasAccess):
		conflict(w, r, "this user already has access to this task")
		return
	case err != nil:
		serverError(w, r, err)
		return
	}

	h.notifyGuest(r.Context(), guest, userID)

	utils.RespondCreated(w, &guest.ID)
}

// requireGuestManagement resolves the {id}/{columnId}/{taskId} route and
// checks that the caller may manage the guests of the task. Writes the
// error and returns false otherwise.
func (h *TaskGuestHandler) requireGuestManagement(w http.ResponseWriter, r *http.Request) (userID, taskID uuid.UUID, ok bool) {
	userID, columnID, access, ok := h.parseTaskContext(w, r)
	if !ok {
		return userID, uuid.Nil, false
	}

	taskID, ok = parseUUIDParam(w, r, "taskId")
	if !ok {
		return userID, taskID, false
	}

	taskExists, err := h.taskRepo.Exists(r.Context(), taskID, columnID)
	if err != nil {
		serverError(w, r, err)
		return userID, taskID, false
	}
	if !taskExists {
		notFound(w, r, "task not found")
		return userID, taskID, false
	}

	if !requirePermission(w, r, permissions.CanInviteGuest(access), errOwnerOrAdminRequired) {
		return userID, taskID, false
	}
	return userID, taskID, true
}

func (h *TaskGuestHandler) parseTaskContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, permissions.Access, bool) {
	userID, workspaceID, access, ok := requireWorkspace(w, r, h.role)
	if !ok {
		return uuid.Nil, uuid.Nil, access, false
	}

	columnID, err := uuid.Parse(chi.URLParam(r, "columnId"))
	if err != nil {
		badRequest(w, r, "invalid columnId")
		return uuid.Nil, uuid.Nil, access, false
	}

	colExists, err := h.columnRepo.Exists(r.Context(), columnID, workspaceID)
	if err != nil {
		serverError(w, r, err)
		return uuid.Nil, uuid.Nil, access, false
	}
	if !colExists {
		notFound(w, r, "column not found")
		return uuid.Nil, uuid.Nil, access, false
	}

	return userID, columnID, access, true
}

// notifyGuest emails the invitation to the guest, on behalf of inviterID.
func (h *TaskGuestHandler) notifyGuest(ctx context.Context, guest models.TaskGuest, inviterID uuid.UUID) {
	inviter, err := h.userRepo.GetByID(ctx, inviterID)
	if err != nil {
		logging.Logger.Error("failed to load inviter for task guest invitation email", slog.Any("error", err))
		return
	}
	h.sendInvitationEmail(guest, inviter)
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
// @Description Expired invitations are included (expires_at in the past) so that they can be resent. Allowed for the organisation owner, or an admin the workspace is shared with.
// @Tags task
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.TaskGuest
// @Failure 400 {object} ErrorResponse "invalid id | invalid columnId | invalid taskId | invalid limit | invalid offset"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found | task not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/guests [get]
func (h *TaskGuestHandler) SentInvitations(w http.ResponseWriter, r *http.Request) {
	_, taskID, ok := h.requireGuestManagement(w, r)
	if !ok {
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

// SetRole godoc
// @Summary Change a guest's role on a task
// @Description Changes the role ('view' or 'edit') of a guest who accepted an invitation on this task. Allowed for the organisation owner, or an admin the workspace is shared with.
// @Tags task
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param userId path string true "Guest's user ID"
// @Param body body setTaskGuestRoleBody true "New role"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} BadRequestResponse "invalid id | invalid columnId | invalid taskId | invalid userId | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found | task not found | guest not found"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/guests/{userId} [put]
func (h *TaskGuestHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	_, taskID, ok := h.requireGuestManagement(w, r)
	if !ok {
		return
	}

	guestID, ok := parseUUIDParam(w, r, "userId")
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[setTaskGuestRoleBody]("TaskGuestHandler.SetRole", w, r)
	if !ok {
		return
	}

	err := h.repo.SetGuestRole(r.Context(), taskID, guestID, body.Role)
	if handleRepoError(w, r, err, "guest not found") {
		return
	}

	utils.RespondUpdated(w)
}

// Remove godoc
// @Summary Remove a guest from a task
// @Description Removes the access of a guest who accepted an invitation on this task; they no longer see it nor appear among its assignees. Allowed for the organisation owner, or an admin the workspace is shared with.
// @Tags task
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param userId path string true "Guest's user ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "invalid id | invalid columnId | invalid taskId | invalid userId"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found | task not found | guest not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/guests/{userId} [delete]
func (h *TaskGuestHandler) Remove(w http.ResponseWriter, r *http.Request) {
	_, taskID, ok := h.requireGuestManagement(w, r)
	if !ok {
		return
	}

	guestID, ok := parseUUIDParam(w, r, "userId")
	if !ok {
		return
	}

	err := h.repo.RemoveGuest(r.Context(), taskID, guestID)
	if handleRepoError(w, r, err, "guest not found") {
		return
	}

	utils.RespondDeleted(w)
}

// Resend godoc
// @Summary Resend a task guest invitation
// @Description Sends the invitation email again and gives the invitation 7 more days from now, expired or not. Allowed for the author of the invitation only, as long as they may still invite guests on the workspace.
// @Tags task
// @Produce json
// @Param id path string true "Invitation ID"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} ErrorResponse "invalid id"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required"
// @Failure 404 {object} ErrorResponse "invitation not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /guest-invitations/{id}/resend [post]
func (h *TaskGuestHandler) Resend(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	invitationID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	invitation, err := h.repo.GetSentInvitation(r.Context(), invitationID, userID)
	if handleRepoError(w, r, err, "invitation not found") {
		return
	}

	access, err := h.role.ResolveAccess(r.Context(), invitation.WorkspaceID, invitation.TaskID, userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if !permissions.CanSeeWorkspace(access) {
		notFound(w, r, "invitation not found")
		return
	}
	if !requirePermission(w, r, permissions.CanInviteGuest(access), errOwnerOrAdminRequired) {
		return
	}

	guest, err := h.repo.Resend(r.Context(), invitationID)
	if handleRepoError(w, r, err, "invitation not found") {
		return
	}

	h.notifyGuest(r.Context(), guest, userID)

	utils.RespondUpdated(w)
}

// ReceivedInvitations godoc
// @Summary List task guest invitations received by the current user
// @Description Returns the pending invitations that have not expired.
// @Tags task
// @Produce json
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.TaskGuest
// @Failure 400 {object} ErrorResponse "invalid limit | invalid offset"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 404 {object} ErrorResponse "user not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
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
// @Description Once accepted, the guest sees the task and appears among its assignees. An expired invitation can no longer be accepted (410); its author may resend it. Only pending invitations can be answered (404 otherwise). 409 if the caller is already a guest of the task, or has joined the task's organisation since the invitation was sent.
// @Tags task
// @Accept json
// @Produce json
// @Param id path string true "Invitation ID"
// @Param body body updateTaskGuestInvitationBody true "New status"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} BadRequestResponse "invalid id | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 404 {object} ErrorResponse "user not found | invitation not found"
// @Failure 409 {object} ErrorResponse "you already have access to this task"
// @Failure 410 {object} ErrorResponse "invitation expired"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
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
	switch {
	case errors.Is(err, repository.ErrTaskGuestInvitationExpired):
		gone(w, r, "invitation expired")
		return
	case errors.Is(err, repository.ErrTaskGuestAlreadyHasAccess):
		conflict(w, r, "you already have access to this task")
		return
	}
	if handleRepoError(w, r, err, "invitation not found") {
		return
	}

	utils.RespondUpdated(w)
}

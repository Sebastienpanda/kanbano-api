package handler

import (
	"context"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/models"
	"kanbano-api/internal/permissions"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/storage"
	"kanbano-api/internal/utils"
	"kanbano-api/internal/ws"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TaskHandler struct {
	repo         *repository.TaskRepository
	columnRepo   *repository.ColumnRepository
	tagRepo      *repository.TagRepository
	role         *repository.RoleRepository
	taskAssignee *repository.TaskAssigneeRepository
	store        *storage.Client
	hub          *ws.Hub
}

type TaskHandlerDeps struct {
	Repo         *repository.TaskRepository
	ColumnRepo   *repository.ColumnRepository
	TagRepo      *repository.TagRepository
	Role         *repository.RoleRepository
	TaskAssignee *repository.TaskAssigneeRepository
	Store        *storage.Client
	Hub          *ws.Hub
}

type createTaskBody struct {
	Name        string     `json:"name" validate:"required,min=1,max=200"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=5000"`
	TagID       *uuid.UUID `json:"tag_id,omitempty"`
	TagName     *string    `json:"tag_name,omitempty" validate:"omitempty,max=50"`
	Status      *string    `json:"status,omitempty" enums:"À faire,En cours,Terminé"`
}

type updateTaskBody struct {
	Name           *string    `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Description    *string    `json:"description,omitempty" validate:"omitempty,max=5000"`
	TagID          *uuid.UUID `json:"tag_id,omitempty"`
	TagName        *string    `json:"tag_name,omitempty" validate:"omitempty,max=50"`
	Status         *string    `json:"status,omitempty" enums:"À faire,En cours,Terminé"`
	Position       *int       `json:"position,omitempty" validate:"omitempty,min=0,max=2147483647"`
	TargetColumnID *uuid.UUID `json:"targetColumnId,omitempty"`
}

type assignTaskBody struct {
	MemberID uuid.UUID `json:"member_id" validate:"required"`
}

// errGuestDescriptionOnly is the 403 of a guest with 'edit' who changes
// anything but the description of their task.
const errGuestDescriptionOnly = "guests can only edit the description"

var validTaskStatuses = map[string]bool{
	"À faire":  true,
	"En cours": true,
	"Terminé":  true,
}

func validateStatus(w http.ResponseWriter, r *http.Request, status *string) bool {
	if status == nil || *status == "" {
		return true
	}
	if !validTaskStatuses[*status] {
		unprocessableEntity(w, r, "invalid status")
		return false
	}
	return true
}

func NewTaskHandler(deps TaskHandlerDeps) *TaskHandler {
	return &TaskHandler{
		repo:         deps.Repo,
		columnRepo:   deps.ColumnRepo,
		tagRepo:      deps.TagRepo,
		role:         deps.Role,
		taskAssignee: deps.TaskAssignee,
		store:        deps.Store,
		hub:          deps.Hub,
	}
}

func (h *TaskHandler) resolveTagID(w http.ResponseWriter, r *http.Request, userID uuid.UUID, tagID *uuid.UUID, tagName *string) (*uuid.UUID, bool) {
	if tagID != nil {
		exists, err := h.tagRepo.Exists(r.Context(), *tagID, userID)
		if err != nil {
			serverError(w, r, err)
			return nil, false
		}
		if !exists {
			notFound(w, r, "tag not found")
			return nil, false
		}
		return tagID, true
	}

	if tagName != nil && *tagName != "" {
		tag, err := h.tagRepo.GetOrCreate(r.Context(), userID, *tagName)
		if err != nil {
			serverError(w, r, err)
			return nil, false
		}
		return &tag.ID, true
	}

	return nil, true
}

func (h *TaskHandler) broadcastTask(ctx context.Context, userID, workspaceID uuid.UUID, eventType ws.EventType, task models.Task) {
	taskWithTag := models.TaskWithTag{
		ID:            task.ID,
		Name:          task.Name,
		Description:   task.Description,
		Position:      task.Position,
		ColumnID:      task.ColumnID,
		TagID:         task.TagID,
		Status:        task.Status,
		CreatedBy:     task.CreatedBy,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
		AssignedUsers: []models.TaskAssignedUser{},
	}
	if task.TagID != nil {
		tag, err := h.tagRepo.GetByID(ctx, *task.TagID)
		if err != nil {
			logging.Logger.Error("failed to load tag for task broadcast", slog.Any("error", err))
		} else {
			taskWithTag.Tag = &tag
		}
	}

	assignees, err := h.taskAssignee.ListForTask(ctx, task.ID)
	if err != nil {
		logging.Logger.Error("failed to load assignees for task broadcast", slog.Any("error", err))
	} else {
		taskWithTag.AssignedUsers = make([]models.TaskAssignedUser, len(assignees))
		for i, a := range assignees {
			taskWithTag.AssignedUsers[i] = models.TaskAssignedUser{
				ID:     a.ID,
				Email:  a.Email,
				Avatar: avatarSet(h.store, a.ID, a.AvatarVersion),
			}
		}
	}

	h.hub.Broadcast(userID, ws.Event{Type: eventType, WorkspaceID: &workspaceID, Data: taskWithTag})
}

// parseTaskContext resolves the workspace like requireWorkspace (including
// the guest role on {taskId} when present), then checks that the
// {columnId} column belongs to it.
func (h *TaskHandler) parseTaskContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, uuid.UUID, permissions.Access, bool) {
	userID, workspaceID, access, ok := requireWorkspace(w, r, h.role)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, access, false
	}

	columnID, err := uuid.Parse(chi.URLParam(r, "columnId"))
	if err != nil {
		badRequest(w, r, "invalid columnId")
		return uuid.Nil, uuid.Nil, uuid.Nil, access, false
	}

	colExists, err := h.columnRepo.Exists(r.Context(), columnID, workspaceID)
	if err != nil {
		serverError(w, r, err)
		return uuid.Nil, uuid.Nil, uuid.Nil, access, false
	}
	if !colExists {
		notFound(w, r, "column not found")
		return uuid.Nil, uuid.Nil, uuid.Nil, access, false
	}

	return userID, workspaceID, columnID, access, true
}

// requireTaskInColumn parses the {taskId} param and checks that the task
// exists in the column. Writes a 400 or 404 and returns false otherwise.
func (h *TaskHandler) requireTaskInColumn(w http.ResponseWriter, r *http.Request, columnID uuid.UUID) (uuid.UUID, bool) {
	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return uuid.Nil, false
	}
	exists, err := h.repo.Exists(r.Context(), taskID, columnID)
	if err != nil {
		serverError(w, r, err)
		return uuid.Nil, false
	}
	if !exists {
		notFound(w, r, "task not found")
		return uuid.Nil, false
	}
	return taskID, true
}

// requireTaskAccess checks that the caller sees the {taskId} task: every
// task for a member with access to the workspace, only the tasks shared
// with them for a guest. Writes a 404 and returns false otherwise.
func requireTaskAccess(w http.ResponseWriter, r *http.Request, access permissions.Access) bool {
	if !permissions.CanViewTask(access) {
		notFound(w, r, "task not found")
		return false
	}
	return true
}

// Create godoc
// @Summary Create a task
// @Tags task
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param body body createTaskBody true "Task to create"
// @Success 201 {object} utils.CreateResponse
// @Failure 400 {object} BadRequestResponse "invalid id | invalid columnId | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found | tag not found"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 422 {object} ErrorResponse "invalid status"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks [post]
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, access, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanCreateOrDeleteContent(access), errOwnerOrAdminRequired) {
		return
	}

	body, ok := utils.DecodeAndValidate[createTaskBody]("TaskHandler.Create", w, r)
	if !ok {
		return
	}
	if !validateStatus(w, r, body.Status) {
		return
	}

	tagID, ok := h.resolveTagID(w, r, userID, body.TagID, body.TagName)
	if !ok {
		return
	}

	task, err := h.repo.Create(r.Context(), body.Name, body.Description, columnID, tagID, body.Status, userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	h.broadcastTask(r.Context(), userID, workspaceID, ws.TaskCreated, task)
	utils.RespondCreated(w, &task.ID)
}

// Update godoc
// @Summary Update or move a task
// @Description Requires edit access on the workspace. A guest invited with 'edit' on the task may only change its description.
// @Tags task
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param body body updateTaskBody true "Fields to update"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} BadRequestResponse "invalid id | invalid columnId | invalid taskId | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "edit access required | guests can only edit the description"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found | task not found | target column not found in this workspace | tag not found"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 422 {object} ErrorResponse "invalid status"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId} [patch]
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, access, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	if !requireTaskAccess(w, r, access) ||
		!requirePermission(w, r, permissions.CanEditTaskDescription(access), errEditAccessRequired) {
		return
	}

	body, ok := utils.DecodeAndValidate[updateTaskBody]("TaskHandler.Update", w, r)
	if !ok {
		return
	}

	onlyDescription := body.Name == nil && body.TagID == nil && body.TagName == nil && body.Status == nil &&
		body.Position == nil && body.TargetColumnID == nil
	if !requirePermission(w, r, permissions.CanEditContent(access) || onlyDescription, errGuestDescriptionOnly) {
		return
	}

	if !validateStatus(w, r, body.Status) {
		return
	}

	if body.TargetColumnID != nil && !h.validateTargetColumn(w, r, workspaceID, *body.TargetColumnID) {
		return
	}

	tagID, ok := h.resolveTagID(w, r, userID, body.TagID, body.TagName)
	if !ok {
		return
	}

	task, err := h.repo.Update(r.Context(), repository.TaskUpdate{
		ID:          taskID,
		ColumnID:    columnID,
		Name:        body.Name,
		Description: body.Description,
		TagID:       tagID,
		Status:      body.Status,
		ActorID:     userID,
	})
	if handleRepoError(w, r, err, "task not found") {
		return
	}

	if body.Position != nil || body.TargetColumnID != nil {
		task, err = h.repo.Reorder(r.Context(), taskID, columnID, body.Position, body.TargetColumnID, userID)
		if handleRepoError(w, r, err, "task not found") {
			return
		}
	}

	h.broadcastTask(r.Context(), userID, workspaceID, ws.TaskUpdated, task)
	utils.RespondUpdated(w)
}

// validateTargetColumn checks that the destination column of a move exists
// in the workspace. Writes a 404 and returns false otherwise.
func (h *TaskHandler) validateTargetColumn(w http.ResponseWriter, r *http.Request, workspaceID, targetColumnID uuid.UUID) bool {
	colExists, err := h.columnRepo.Exists(r.Context(), targetColumnID, workspaceID)
	if err != nil {
		serverError(w, r, err)
		return false
	}
	if !colExists {
		notFound(w, r, "target column not found in this workspace")
		return false
	}

	return true
}

// Delete godoc
// @Summary Delete a task
// @Description Soft-deletes a task.
// @Tags task
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "invalid id | invalid columnId | invalid taskId"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found | task not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId} [delete]
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, access, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	if !requireTaskAccess(w, r, access) ||
		!requirePermission(w, r, permissions.CanCreateOrDeleteContent(access), errOwnerOrAdminRequired) {
		return
	}

	task, err := h.repo.SoftDelete(r.Context(), taskID, columnID, userID)
	if handleRepoError(w, r, err, "task not found") {
		return
	}

	h.broadcastTask(r.Context(), userID, workspaceID, ws.TaskDeleted, task)

	utils.RespondDeleted(w)
}

// Assignees godoc
// @Summary List a task's assignees
// @Description Members assigned to the task, followed by the guests who accepted an invitation on it (is_guest).
// @Tags task
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Success 200 {array} models.TaskAssignee
// @Failure 400 {object} ErrorResponse "invalid id | invalid columnId | invalid taskId"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found | task not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/assignees [get]
func (h *TaskHandler) Assignees(w http.ResponseWriter, r *http.Request) {
	_, _, columnID, access, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := h.requireTaskInColumn(w, r, columnID)
	if !ok {
		return
	}

	if !requireTaskAccess(w, r, access) {
		return
	}

	assignees, err := h.taskAssignee.ListForTask(r.Context(), taskID)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, assignees)
}

// Assign godoc
// @Summary Assign a member to a task
// @Description Shows the member as working on the task. An assignment grants no access and changes no role. The member must have access to the workspace. Requires edit access on the workspace.
// @Tags task
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param body body assignTaskBody true "Member to assign"
// @Success 204 "No Content"
// @Failure 400 {object} BadRequestResponse "invalid id | invalid columnId | invalid taskId | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "edit access required"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found | task not found | member not found in this workspace"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/assignees [post]
func (h *TaskHandler) Assign(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, access, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := h.requireTaskInColumn(w, r, columnID)
	if !ok {
		return
	}

	if !requireTaskAccess(w, r, access) ||
		!requirePermission(w, r, permissions.CanAssign(access), errEditAccessRequired) {
		return
	}

	body, ok := utils.DecodeAndValidate[assignTaskBody]("TaskHandler.Assign", w, r)
	if !ok {
		return
	}

	memberAccess, err := h.role.ResolveAccess(r.Context(), workspaceID, uuid.Nil, body.MemberID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if !permissions.CanViewWorkspace(memberAccess) {
		notFound(w, r, "member not found in this workspace")
		return
	}

	if err := h.taskAssignee.Assign(r.Context(), taskID, body.MemberID, userID); err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondDeleted(w)
}

// Unassign godoc
// @Summary Unassign a member from a task
// @Tags task
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param memberId path string true "Member ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "invalid id | invalid columnId | invalid taskId | invalid memberId"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "edit access required"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found | task not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/assignees/{memberId} [delete]
func (h *TaskHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	_, _, columnID, access, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := h.requireTaskInColumn(w, r, columnID)
	if !ok {
		return
	}

	if !requireTaskAccess(w, r, access) ||
		!requirePermission(w, r, permissions.CanAssign(access), errEditAccessRequired) {
		return
	}

	memberID, ok := parseUUIDParam(w, r, "memberId")
	if !ok {
		return
	}

	if err := h.taskAssignee.Unassign(r.Context(), taskID, memberID); err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondDeleted(w)
}

// Roles godoc
// @Summary List the caller's resolved role for every task in a workspace
// @Description Lists the tasks the caller sees, with their role on each: every task with the same role for a member ('edit' if they may modify tasks), only the tasks shared with them for a guest, with their guest role ('edit' then only allows changing the description).
// @Tags task
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 200 {array} models.TaskRole
// @Failure 400 {object} ErrorResponse "invalid id"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 404 {object} ErrorResponse "workspace not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/tasks/roles [get]
func (h *TaskHandler) Roles(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, access, ok := requireWorkspace(w, r, h.role)
	if !ok {
		return
	}

	roles, err := h.role.ListTaskRoles(r.Context(), workspaceID, userID, access)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, roles)
}

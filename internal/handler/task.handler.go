package handler

import (
	"context"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/models"
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
	repo          *repository.TaskRepository
	workspaceRepo *repository.WorkspaceRepository
	columnRepo    *repository.ColumnRepository
	tagRepo       *repository.TagRepository
	accessGrant   *repository.AccessGrantRepository
	taskAssignee  *repository.TaskAssigneeRepository
	store         *storage.Client
	hub           *ws.Hub
}

type TaskHandlerDeps struct {
	Repo          *repository.TaskRepository
	WorkspaceRepo *repository.WorkspaceRepository
	ColumnRepo    *repository.ColumnRepository
	TagRepo       *repository.TagRepository
	AccessGrant   *repository.AccessGrantRepository
	TaskAssignee  *repository.TaskAssigneeRepository
	Store         *storage.Client
	Hub           *ws.Hub
}

type createTaskBody struct {
	Name        string     `json:"name" validate:"required,min=1,max=200"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=5000"`
	TagID       *uuid.UUID `json:"tag_id,omitempty"`
	TagName     *string    `json:"tag_name,omitempty" validate:"omitempty,max=50"`
	Status      *string    `json:"status,omitempty"`
}

type updateTaskBody struct {
	Name           *string    `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Description    *string    `json:"description,omitempty" validate:"omitempty,max=5000"`
	TagID          *uuid.UUID `json:"tag_id,omitempty"`
	TagName        *string    `json:"tag_name,omitempty" validate:"omitempty,max=50"`
	Status         *string    `json:"status,omitempty"`
	Position       *int       `json:"position,omitempty" validate:"omitempty,min=0"`
	TargetColumnID *uuid.UUID `json:"targetColumnId,omitempty"`
}

type assignTaskBody struct {
	MemberID uuid.UUID `json:"member_id" validate:"required"`
	Role     string    `json:"role" validate:"required,oneof=view edit"`
}

var validTaskStatuses = map[string]bool{
	"À faire":  true,
	"En cours": true,
	"Terminé":  true,
}

func validateStatus(w http.ResponseWriter, status *string) bool {
	if status == nil || *status == "" {
		return true
	}
	if !validTaskStatuses[*status] {
		unprocessableEntity(w, "invalid status")
		return false
	}
	return true
}

func NewTaskHandler(deps TaskHandlerDeps) *TaskHandler {
	return &TaskHandler{
		repo:          deps.Repo,
		workspaceRepo: deps.WorkspaceRepo,
		columnRepo:    deps.ColumnRepo,
		tagRepo:       deps.TagRepo,
		accessGrant:   deps.AccessGrant,
		taskAssignee:  deps.TaskAssignee,
		store:         deps.Store,
		hub:           deps.Hub,
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
			notFound(w, "tag not found")
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

func (h *TaskHandler) parseTaskContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	userID, workspaceID, ok := requireWorkspace(w, r, h.workspaceRepo)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	columnID, err := uuid.Parse(chi.URLParam(r, "columnId"))
	if err != nil {
		badRequest(w, "invalid columnId")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	colExists, err := h.columnRepo.Exists(r.Context(), columnID, workspaceID)
	if err != nil {
		serverError(w, r, err)
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if !colExists {
		notFound(w, "column not found")
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}

	return userID, workspaceID, columnID, true
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
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks [post]
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	hasAccess, err := h.accessGrant.HasColumnAccess(r.Context(), workspaceID, columnID, userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if !hasAccess {
		notFound(w, "column not found")
		return
	}

	body, ok := utils.DecodeAndValidate[createTaskBody]("TaskHandler.Create", w, r)
	if !ok {
		return
	}
	if !validateStatus(w, body.Status) {
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

func (h *TaskHandler) requireTaskAccess(w http.ResponseWriter, r *http.Request, workspaceID, columnID, taskID, userID uuid.UUID) bool {
	hasColumnAccess, err := h.accessGrant.HasColumnAccess(r.Context(), workspaceID, columnID, userID)
	if err != nil {
		serverError(w, r, err)
		return false
	}
	if hasColumnAccess {
		return true
	}

	isAssigned, err := h.taskAssignee.HasAccess(r.Context(), taskID, userID)
	if err != nil {
		serverError(w, r, err)
		return false
	}
	if !isAssigned {
		notFound(w, "task not found")
		return false
	}
	return true
}

// Update godoc
// @Summary Update or move a task
// @Tags task
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param body body updateTaskBody true "Fields to update"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId} [patch]
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	if !h.requireTaskAccess(w, r, workspaceID, columnID, taskID, userID) {
		return
	}

	body, ok := utils.DecodeAndValidate[updateTaskBody]("TaskHandler.Update", w, r)
	if !ok {
		return
	}
	if !validateStatus(w, body.Status) {
		return
	}

	if body.TargetColumnID != nil && !h.validateTargetColumn(w, r, workspaceID, *body.TargetColumnID, userID) {
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

// validateTargetColumn vérifie que la colonne de destination d'un déplacement
// existe dans le workspace et est accessible à l'utilisateur. Écrit une
// réponse d'erreur et retourne false si l'une des vérifications échoue.
func (h *TaskHandler) validateTargetColumn(w http.ResponseWriter, r *http.Request, workspaceID, targetColumnID, userID uuid.UUID) bool {
	colExists, err := h.columnRepo.Exists(r.Context(), targetColumnID, workspaceID)
	if err != nil {
		serverError(w, r, err)
		return false
	}
	if !colExists {
		notFound(w, "target column not found in this workspace")
		return false
	}

	hasTargetAccess, err := h.accessGrant.HasColumnAccess(r.Context(), workspaceID, targetColumnID, userID)
	if err != nil {
		serverError(w, r, err)
		return false
	}
	if !hasTargetAccess {
		notFound(w, "target column not found in this workspace")
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
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId} [delete]
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	if !h.requireTaskAccess(w, r, workspaceID, columnID, taskID, userID) {
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
// @Tags task
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Success 200 {array} models.TaskAssignee
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/assignees [get]
func (h *TaskHandler) Assignees(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	if !h.requireTaskAccess(w, r, workspaceID, columnID, taskID, userID) {
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
// @Description The member must already have access to the task's column.
// @Tags task
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param taskId path string true "Task ID"
// @Param body body assignTaskBody true "Member to assign"
// @Success 204 "No Content"
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/assignees [post]
func (h *TaskHandler) Assign(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	if !h.requireTaskAccess(w, r, workspaceID, columnID, taskID, userID) {
		return
	}

	body, ok := utils.DecodeAndValidate[assignTaskBody]("TaskHandler.Assign", w, r)
	if !ok {
		return
	}

	hasAccess, err := h.accessGrant.HasColumnAccess(r.Context(), workspaceID, columnID, body.MemberID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if !hasAccess {
		notFound(w, "member not found in this column")
		return
	}

	if err := h.taskAssignee.Assign(r.Context(), taskID, body.MemberID, userID, body.Role); err != nil {
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
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId}/tasks/{taskId}/assignees/{memberId} [delete]
func (h *TaskHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	if !h.requireTaskAccess(w, r, workspaceID, columnID, taskID, userID) {
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

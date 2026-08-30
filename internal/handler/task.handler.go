package handler

import (
	"context"
	"kanbano-api/internal/models"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/ws"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TaskHandler struct {
	repo          *repository.TaskRepository
	workspaceRepo *repository.WorkspaceRepository
	columnRepo    *repository.ColumnRepository
	tagRepo       *repository.TagRepository
	hub           *ws.Hub
}

type createTaskBody struct {
	Name        string     `json:"name" validate:"required,min=1,max=200"`
	Description *string    `json:"description" validate:"omitempty,max=5000"`
	TagID       *uuid.UUID `json:"tag_id"`
	TagName     *string    `json:"tag_name" validate:"omitempty,max=50"`
	Status      *string    `json:"status"`
}

type updateTaskBody struct {
	Name        *string    `json:"name" validate:"omitempty,min=1,max=200"`
	Description *string    `json:"description" validate:"omitempty,max=5000"`
	TagID       *uuid.UUID `json:"tag_id"`
	TagName     *string    `json:"tag_name" validate:"omitempty,max=50"`
	Status      *string    `json:"status"`
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
		http.Error(w, "invalid status", http.StatusBadRequest)
		return false
	}
	return true
}

type reorderTaskBody struct {
	Position       *int       `json:"position" validate:"omitempty,min=0"`
	TargetColumnID *uuid.UUID `json:"targetColumnId"`
}

func NewTaskHandler(repo *repository.TaskRepository, workspaceRepo *repository.WorkspaceRepository, columnRepo *repository.ColumnRepository, tagRepo *repository.TagRepository, hub *ws.Hub) *TaskHandler {
	return &TaskHandler{repo: repo, workspaceRepo: workspaceRepo, columnRepo: columnRepo, tagRepo: tagRepo, hub: hub}
}

func (h *TaskHandler) resolveTagID(w http.ResponseWriter, r *http.Request, userID uuid.UUID, tagID *uuid.UUID, tagName *string) (*uuid.UUID, bool) {
	if tagID != nil {
		exists, err := h.tagRepo.Exists(r.Context(), *tagID, userID)
		if err != nil {
			serverError(w, err)
			return nil, false
		}
		if !exists {
			http.Error(w, "tag not found", http.StatusNotFound)
			return nil, false
		}
		return tagID, true
	}

	if tagName != nil && *tagName != "" {
		tag, err := h.tagRepo.GetOrCreate(r.Context(), userID, *tagName)
		if err != nil {
			serverError(w, err)
			return nil, false
		}
		return &tag.ID, true
	}

	return nil, true
}

func (h *TaskHandler) broadcastTask(ctx context.Context, userID, workspaceID uuid.UUID, eventType ws.EventType, task models.Task) {
	taskWithTag := models.TaskWithTag{Task: task}
	if task.TagID != nil {
		tag, err := h.tagRepo.GetByID(ctx, *task.TagID)
		if err != nil {
			log.Printf("failed to load tag for task broadcast: %v", err)
		} else {
			taskWithTag.Tag = &tag
		}
	}
	h.hub.Broadcast(userID, ws.Event{Type: eventType, WorkspaceID: workspaceID, Data: taskWithTag})
}

func (h *TaskHandler) parseTaskContext(w http.ResponseWriter, r *http.Request) (userID, workspaceID, columnID uuid.UUID, ok bool) {
	userID, workspaceID, ok = requireWorkspace(w, r, h.workspaceRepo)
	if !ok {
		return
	}

	columnID, err := uuid.Parse(chi.URLParam(r, "columnId"))
	if err != nil {
		http.Error(w, "invalid columnId", http.StatusBadRequest)
		ok = false
		return
	}

	colExists, err := h.columnRepo.Exists(r.Context(), columnID, workspaceID)
	if err != nil {
		serverError(w, err)
		ok = false
		return
	}
	if !colExists {
		http.Error(w, "column not found", http.StatusNotFound)
		ok = false
		return
	}

	ok = true
	return
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	body, ok := decodeAndValidate[createTaskBody](w, r)
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

	task, err := h.repo.Create(r.Context(), body.Name, body.Description, columnID, tagID, body.Status)
	if err == nil {
		h.broadcastTask(r.Context(), userID, workspaceID, ws.TaskCreated, task)
	}
	respondCreated(w, task, err)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[updateTaskBody](w, r)
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

	task, err := h.repo.Update(r.Context(), taskID, columnID, body.Name, body.Description, tagID, body.Status)
	if handleRepoError(w, err, "task not found") {
		return
	}

	h.broadcastTask(r.Context(), userID, workspaceID, ws.TaskUpdated, task)
	writeJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	_, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	body, ok := decodeAndValidate[reorderTaskBody](w, r)
	if !ok {
		return
	}

	if body.TargetColumnID == nil && body.Position == nil {
		http.Error(w, "position or targetColumnId required", http.StatusBadRequest)
		return
	}

	if body.TargetColumnID != nil {
		colExists, err := h.columnRepo.Exists(r.Context(), *body.TargetColumnID, workspaceID)
		if err != nil {
			serverError(w, err)
			return
		}
		if !colExists {
			http.Error(w, "target column not found in this workspace", http.StatusNotFound)
			return
		}
	}

	task, err := h.repo.Reorder(r.Context(), taskID, columnID, body.Position, body.TargetColumnID)
	if handleRepoError(w, err, "task not found") {
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	task, err := h.repo.SoftDelete(r.Context(), taskID, columnID)
	if handleRepoError(w, err, "task not found") {
		return
	}

	h.broadcastTask(r.Context(), userID, workspaceID, ws.TaskDeleted, task)

	writeJSON(w, http.StatusOK, task)
}

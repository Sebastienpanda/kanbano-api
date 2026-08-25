package handler

import (
	"kanbano-api/internal/repository"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TaskHandler struct {
	repo          *repository.TaskRepository
	workspaceRepo *repository.WorkspaceRepository
	columnRepo    *repository.ColumnRepository
	stateRepo     *repository.StateRepository
}

type createTaskBody struct {
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	StateID     *uuid.UUID `json:"state_id"`
	StateName   *string    `json:"state_name"`
}

type updateTaskBody struct {
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	StateID     *uuid.UUID `json:"state_id"`
	StateName   *string    `json:"state_name"`
}

type reorderTaskBody struct {
	Position       *int       `json:"position"`
	TargetColumnID *uuid.UUID `json:"targetColumnId"`
}

func NewTaskHandler(repo *repository.TaskRepository, workspaceRepo *repository.WorkspaceRepository, columnRepo *repository.ColumnRepository, stateRepo *repository.StateRepository) *TaskHandler {
	return &TaskHandler{repo: repo, workspaceRepo: workspaceRepo, columnRepo: columnRepo, stateRepo: stateRepo}
}

func (h *TaskHandler) resolveStateID(w http.ResponseWriter, r *http.Request, userID uuid.UUID, stateID *uuid.UUID, stateName *string) (*uuid.UUID, bool) {
	if stateID != nil {
		exists, err := h.stateRepo.Exists(r.Context(), *stateID, userID)
		if err != nil {
			serverError(w, err)
			return nil, false
		}
		if !exists {
			http.Error(w, "state not found", http.StatusNotFound)
			return nil, false
		}
		return stateID, true
	}

	if stateName != nil && *stateName != "" {
		state, err := h.stateRepo.GetOrCreate(r.Context(), userID, *stateName)
		if err != nil {
			serverError(w, err)
			return nil, false
		}
		return &state.ID, true
	}

	return nil, true
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
	userID, _, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	body, ok := decodeJSON[createTaskBody](w, r)
	if !ok {
		return
	}
	if body.Name == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	stateID, ok := h.resolveStateID(w, r, userID, body.StateID, body.StateName)
	if !ok {
		return
	}

	task, err := h.repo.Create(r.Context(), body.Name, body.Description, columnID, stateID)
	respondCreated(w, task, err)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, ok := parseUUIDParam(w, r, "taskId")
	if !ok {
		return
	}

	body, ok := decodeJSON[updateTaskBody](w, r)
	if !ok {
		return
	}

	stateID, ok := h.resolveStateID(w, r, userID, body.StateID, body.StateName)
	if !ok {
		return
	}

	task, err := h.repo.Update(r.Context(), taskID, columnID, body.Name, body.Description, stateID)
	if handleRepoError(w, err, "task not found") {
		return
	}

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

	body, ok := decodeJSON[reorderTaskBody](w, r)
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
	_, _, columnID, ok := h.parseTaskContext(w, r)
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

	writeJSON(w, http.StatusOK, task)
}

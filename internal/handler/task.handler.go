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

// resolveStateID détermine le state_id à appliquer : stateID s'il est fourni, sinon
// stateName crée (ou récupère) le statut correspondant pour le user.
func (h *TaskHandler) resolveStateID(w http.ResponseWriter, r *http.Request, userID uuid.UUID, stateID *uuid.UUID, stateName *string) (*uuid.UUID, bool) {
	if stateID != nil {
		exists, err := h.stateRepo.Exists(r.Context(), *stateID, userID)
		if err != nil {
			serverError(w, err)
			return nil, false
		}
		if !exists {
			http.Error(w, "statut introuvable", http.StatusNotFound)
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
	userID = userIDFromContext(r)

	workspaceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "id invalide", http.StatusBadRequest)
		return
	}

	columnID, err = uuid.Parse(chi.URLParam(r, "columnId"))
	if err != nil {
		http.Error(w, "columnId invalide", http.StatusBadRequest)
		return
	}

	exists, err := h.workspaceRepo.Exists(r.Context(), workspaceID, userID)
	if err != nil || !exists {
		http.Error(w, "workspace introuvable", http.StatusNotFound)
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
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	stateID, ok := h.resolveStateID(w, r, userID, body.StateID, body.StateName)
	if !ok {
		return
	}

	task, err := h.repo.Create(r.Context(), body.Name, body.Description, columnID, stateID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, task)
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
	if handleRepoError(w, err, "tâche introuvable") {
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// Reorder déplace une tâche à une nouvelle position, éventuellement vers une autre colonne
// du même workspace (targetColumnId). Le back décale automatiquement les autres tâches.
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
		http.Error(w, "position ou targetColumnId requis", http.StatusBadRequest)
		return
	}

	if body.TargetColumnID != nil {
		colExists, err := h.columnRepo.Exists(r.Context(), *body.TargetColumnID, workspaceID)
		if err != nil {
			serverError(w, err)
			return
		}
		if !colExists {
			http.Error(w, "colonne cible introuvable dans ce workspace", http.StatusNotFound)
			return
		}
	}

	task, err := h.repo.Reorder(r.Context(), taskID, columnID, body.Position, body.TargetColumnID)
	if handleRepoError(w, err, "tâche introuvable") {
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
	if handleRepoError(w, err, "tâche introuvable") {
		return
	}

	writeJSON(w, http.StatusOK, task)
}

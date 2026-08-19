package handler

import (
	"encoding/json"
	"errors"
	"kanbano-api/internal/repository"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TaskHandler struct {
	repo          *repository.TaskRepository
	workspaceRepo *repository.WorkspaceRepository
	columnRepo    *repository.ColumnRepository
}

func NewTaskHandler(repo *repository.TaskRepository, workspaceRepo *repository.WorkspaceRepository, columnRepo *repository.ColumnRepository) *TaskHandler {
	return &TaskHandler{repo: repo, workspaceRepo: workspaceRepo, columnRepo: columnRepo}
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
	_, _, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(r.Context(), body.Name, body.Description, columnID)
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

	taskID, err := uuid.Parse(chi.URLParam(r, "taskId"))
	if err != nil {
		http.Error(w, "taskId invalide", http.StatusBadRequest)
		return
	}

	var body struct {
		Name           *string    `json:"name"`
		Description    *string    `json:"description"`
		Position       *int       `json:"position"`
		WorkspaceID    *uuid.UUID `json:"workspaceId"`
		TargetColumnID *uuid.UUID `json:"targetColumnId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	if (body.WorkspaceID == nil) != (body.TargetColumnID == nil) {
		http.Error(w, "workspaceId et targetColumnId doivent être fournis ensemble", http.StatusBadRequest)
		return
	}

	if body.WorkspaceID != nil {
		wsExists, err := h.workspaceRepo.Exists(r.Context(), *body.WorkspaceID, userID)
		if err != nil {
			serverError(w, err)
			return
		}
		if !wsExists {
			http.Error(w, "workspace cible introuvable", http.StatusNotFound)
			return
		}

		colExists, err := h.columnRepo.Exists(r.Context(), *body.TargetColumnID, *body.WorkspaceID)
		if err != nil {
			serverError(w, err)
			return
		}
		if !colExists {
			http.Error(w, "colonne cible introuvable dans ce workspace", http.StatusNotFound)
			return
		}
	}

	task, err := h.repo.Update(r.Context(), taskID, columnID, body.Name, body.Description, body.Position, body.TargetColumnID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "tâche introuvable", http.StatusNotFound)
			return
		}
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	_, _, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "taskId"))
	if err != nil {
		http.Error(w, "taskId invalide", http.StatusBadRequest)
		return
	}

	task, err := h.repo.SoftDelete(r.Context(), taskID, columnID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "tâche introuvable", http.StatusNotFound)
			return
		}
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

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
}

func NewTaskHandler(repo *repository.TaskRepository, workspaceRepo *repository.WorkspaceRepository) *TaskHandler {
	return &TaskHandler{repo: repo, workspaceRepo: workspaceRepo}
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
		Title       string  `json:"title"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(r.Context(), body.Title, body.Description, columnID)
	if err != nil {
		http.Error(w, "erreur serveur", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	_, _, columnID, ok := h.parseTaskContext(w, r)
	if !ok {
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "taskId"))
	if err != nil {
		http.Error(w, "taskId invalide", http.StatusBadRequest)
		return
	}

	var body struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Position    *int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	task, err := h.repo.Update(r.Context(), taskID, columnID, body.Title, body.Description, body.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "tâche introuvable", http.StatusNotFound)
			return
		}
		http.Error(w, "erreur serveur", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

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

type WorkspaceHandler struct {
	repo *repository.WorkspaceRepository
}

func NewWorkspaceHandler(repo *repository.WorkspaceRepository) *WorkspaceHandler {
	return &WorkspaceHandler{repo: repo}
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaces, err := h.repo.List(r.Context(), userID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, workspaces)
}

func (h *WorkspaceHandler) Recent(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaces, err := h.repo.ListRecent(r.Context(), userID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, workspaces)
}

func (h *WorkspaceHandler) Names(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	names, err := h.repo.ListNames(r.Context(), userID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, names)
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	workspace, err := h.repo.Create(r.Context(), body.Name, body.Description, userID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, workspace)
}

func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "id invalide", http.StatusBadRequest)
		return
	}

	detail, err := h.repo.GetByID(r.Context(), workspaceID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "workspace introuvable", http.StatusNotFound)
			return
		}
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "id invalide", http.StatusBadRequest)
		return
	}

	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	workspace, err := h.repo.Update(r.Context(), workspaceID, userID, body.Name, body.Description)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "workspace introuvable", http.StatusNotFound)
			return
		}
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, workspace)
}

func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "id invalide", http.StatusBadRequest)
		return
	}

	workspace, err := h.repo.SoftDelete(r.Context(), workspaceID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "workspace introuvable", http.StatusNotFound)
			return
		}
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, workspace)
}

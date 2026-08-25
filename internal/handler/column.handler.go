package handler

import (
	"errors"
	"kanbano-api/internal/repository"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type ColumnHandler struct {
	repo          *repository.ColumnRepository
	workspaceRepo *repository.WorkspaceRepository
}

type createColumnBody struct {
	Name string `json:"name"`
}

type updateColumnBody struct {
	Name *string `json:"name"`
}

type reorderColumnBody struct {
	Position int `json:"position"`
}

func NewColumnHandler(repo *repository.ColumnRepository, workspaceRepo *repository.WorkspaceRepository) *ColumnHandler {
	return &ColumnHandler{repo: repo, workspaceRepo: workspaceRepo}
}

func (h *ColumnHandler) NamesByWorkspaceName(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "paramètre name manquant", http.StatusBadRequest)
		return
	}

	workspaceID, err := h.workspaceRepo.GetIDByName(r.Context(), name, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "workspace introuvable", http.StatusNotFound)
			return
		}
		serverError(w, err)
		return
	}

	names, err := h.repo.ListNames(r.Context(), workspaceID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, names)
}

func (h *ColumnHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	exists, err := h.workspaceRepo.Exists(r.Context(), workspaceID, userID)
	if err != nil || !exists {
		http.Error(w, "workspace introuvable", http.StatusNotFound)
		return
	}

	body, ok := decodeJSON[createColumnBody](w, r)
	if !ok {
		return
	}
	if body.Name == "" {
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	column, err := h.repo.Create(r.Context(), body.Name, workspaceID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, column)
}

func (h *ColumnHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	columnID, ok := parseUUIDParam(w, r, "columnId")
	if !ok {
		return
	}

	exists, err := h.workspaceRepo.Exists(r.Context(), workspaceID, userID)
	if err != nil || !exists {
		http.Error(w, "workspace introuvable", http.StatusNotFound)
		return
	}

	body, ok := decodeJSON[updateColumnBody](w, r)
	if !ok {
		return
	}

	column, err := h.repo.Update(r.Context(), columnID, workspaceID, body.Name)
	if handleRepoError(w, err, "colonne introuvable") {
		return
	}

	writeJSON(w, http.StatusOK, column)
}

func (h *ColumnHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	columnID, ok := parseUUIDParam(w, r, "columnId")
	if !ok {
		return
	}

	exists, err := h.workspaceRepo.Exists(r.Context(), workspaceID, userID)
	if err != nil || !exists {
		http.Error(w, "workspace introuvable", http.StatusNotFound)
		return
	}

	body, ok := decodeJSON[reorderColumnBody](w, r)
	if !ok {
		return
	}

	column, err := h.repo.Reorder(r.Context(), columnID, workspaceID, body.Position)
	if handleRepoError(w, err, "colonne introuvable") {
		return
	}

	writeJSON(w, http.StatusOK, column)
}

func (h *ColumnHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}
	columnID, ok := parseUUIDParam(w, r, "columnId")
	if !ok {
		return
	}

	exists, err := h.workspaceRepo.Exists(r.Context(), workspaceID, userID)
	if err != nil || !exists {
		http.Error(w, "workspace introuvable", http.StatusNotFound)
		return
	}

	column, err := h.repo.SoftDelete(r.Context(), columnID, workspaceID)
	if handleRepoError(w, err, "colonne introuvable") {
		return
	}

	writeJSON(w, http.StatusOK, column)
}

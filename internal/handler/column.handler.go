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

type ColumnHandler struct {
	repo          *repository.ColumnRepository
	workspaceRepo *repository.WorkspaceRepository
}

func NewColumnHandler(repo *repository.ColumnRepository, workspaceRepo *repository.WorkspaceRepository) *ColumnHandler {
	return &ColumnHandler{repo: repo, workspaceRepo: workspaceRepo}
}

func (h *ColumnHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "id invalide", http.StatusBadRequest)
		return
	}

	exists, err := h.workspaceRepo.Exists(r.Context(), workspaceID, userID)
	if err != nil || !exists {
		http.Error(w, "workspace introuvable", http.StatusNotFound)
		return
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	column, err := h.repo.Create(r.Context(), body.Title, workspaceID)
	if err != nil {
		http.Error(w, "erreur serveur", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, column)
}

func (h *ColumnHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "id invalide", http.StatusBadRequest)
		return
	}

	columnID, err := uuid.Parse(chi.URLParam(r, "columnId"))
	if err != nil {
		http.Error(w, "columnId invalide", http.StatusBadRequest)
		return
	}

	exists, err := h.workspaceRepo.Exists(r.Context(), workspaceID, userID)
	if err != nil || !exists {
		http.Error(w, "workspace introuvable", http.StatusNotFound)
		return
	}

	var body struct {
		Title    *string `json:"title"`
		Position *int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	column, err := h.repo.Update(r.Context(), columnID, workspaceID, body.Title, body.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "colonne introuvable", http.StatusNotFound)
			return
		}
		http.Error(w, "erreur serveur", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, column)
}

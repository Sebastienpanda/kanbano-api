package handler

import (
	"kanbano-api/internal/repository"
	"net/http"
)

type WorkspaceHandler struct {
	repo *repository.WorkspaceRepository
}

type createWorkspaceBody struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type updateWorkspaceBody struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
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

	body, ok := decodeJSON[createWorkspaceBody](w, r)
	if !ok {
		return
	}
	if body.Name == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	workspace, err := h.repo.Create(r.Context(), body.Name, body.Description, userID)
	respondCreated(w, workspace, err)
}

func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	detail, err := h.repo.GetByID(r.Context(), workspaceID, userID)
	if handleRepoError(w, err, "workspace not found") {
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeJSON[updateWorkspaceBody](w, r)
	if !ok {
		return
	}

	workspace, err := h.repo.Update(r.Context(), workspaceID, userID, body.Name, body.Description)
	if handleRepoError(w, err, "workspace not found") {
		return
	}

	writeJSON(w, http.StatusOK, workspace)
}

func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	workspace, err := h.repo.SoftDelete(r.Context(), workspaceID, userID)
	if handleRepoError(w, err, "workspace not found") {
		return
	}

	writeJSON(w, http.StatusOK, workspace)
}

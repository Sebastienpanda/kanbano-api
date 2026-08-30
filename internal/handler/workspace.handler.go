package handler

import (
	"context"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/utils"
	"kanbano-api/internal/ws"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type WorkspaceHandler struct {
	repo *repository.WorkspaceRepository
	hub  *ws.Hub
}

type createWorkspaceBody struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
}

type updateWorkspaceBody struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
}

func NewWorkspaceHandler(repo *repository.WorkspaceRepository, hub *ws.Hub) *WorkspaceHandler {
	return &WorkspaceHandler{repo: repo, hub: hub}
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaces, err := h.repo.List(r.Context(), userID)
	if err != nil {
		serverError(w, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, workspaces)
}

func (h *WorkspaceHandler) Recent(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaces, err := h.repo.ListRecent(r.Context(), userID)
	if err != nil {
		serverError(w, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, workspaces)
}

func (h *WorkspaceHandler) Names(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	names, err := h.repo.ListNames(r.Context(), userID)
	if err != nil {
		serverError(w, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, names)
}

func (h *WorkspaceHandler) Search(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		badRequest(w, "missing query parameter q")
		return
	}

	results, err := h.repo.Search(r.Context(), userID, query)
	if err != nil {
		serverError(w, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, results)
}

func (h *WorkspaceHandler) recentOrNil(ctx context.Context, userID uuid.UUID) any {
	recent, err := h.repo.ListRecent(ctx, userID)
	if err != nil {
		log.Printf("failed to load recent workspaces for broadcast: %v", err)
		return nil
	}
	return recent
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	body, ok := utils.DecodeAndValidate[createWorkspaceBody]("WorkspaceHandler.Create", w, r)
	if !ok {
		return
	}

	workspace, err := h.repo.Create(r.Context(), body.Name, body.Description, userID)
	if err != nil {
		serverError(w, err)
		return
	}
	h.hub.Broadcast(userID, ws.Event{
		Type:        ws.WorkspaceCreated,
		WorkspaceID: workspace.ID,
		Data:        workspace,
		Recent:      h.recentOrNil(r.Context(), userID),
	})
	utils.RespondCreated(w, &workspace.ID)
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

	utils.RespondJSON(w, http.StatusOK, detail)
}

func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[updateWorkspaceBody]("WorkspaceHandler.Update", w, r)
	if !ok {
		return
	}

	workspace, err := h.repo.Update(r.Context(), workspaceID, userID, body.Name, body.Description)
	if handleRepoError(w, err, "workspace not found") {
		return
	}

	h.hub.Broadcast(userID, ws.Event{Type: ws.WorkspaceUpdated, WorkspaceID: workspace.ID, Data: workspace})
	utils.RespondUpdated(w)
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

	h.hub.Broadcast(userID, ws.Event{
		Type:        ws.WorkspaceDeleted,
		WorkspaceID: workspace.ID,
	})
	utils.RespondDeleted(w)
}

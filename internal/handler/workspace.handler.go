package handler

import (
	"context"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/models"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/storage"
	"kanbano-api/internal/utils"
	"kanbano-api/internal/ws"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type WorkspaceHandler struct {
	repo  *repository.WorkspaceRepository
	store *storage.Client
	hub   *ws.Hub
}

type createWorkspaceBody struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`
}

type updateWorkspaceBody struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`
}

func NewWorkspaceHandler(repo *repository.WorkspaceRepository, store *storage.Client, hub *ws.Hub) *WorkspaceHandler {
	return &WorkspaceHandler{repo: repo, store: store, hub: hub}
}

func (h *WorkspaceHandler) resolveAssigneeAvatars(detail *models.WorkspaceDetail) {
	for c := range detail.Columns {
		for t := range detail.Columns[c].Tasks {
			users := detail.Columns[c].Tasks[t].AssignedUsers
			for u := range users {
				users[u].Avatar = avatarSet(h.store, users[u].ID, users[u].AvatarVersion)
			}
		}
	}
}

// List godoc
// @Summary List workspaces
// @Description List the current user's workspaces. The response shape depends on which query params are set (checked in this order): 1) q set -> returns []models.WorkspaceSearchResult (search results); 2) view=names -> returns []models.WorkspaceName (id+name only); 3) view=recent -> returns []models.Workspace (most recently accessed, unpaginated); 4) none of the above -> returns []models.Workspace, paginated via limit/offset. Only one variant applies per request; q takes priority over view.
// @Tags workspace
// @Produce json
// @Param q query string false "Search query. When set, returns []models.WorkspaceSearchResult regardless of other params."
// @Param view query string false "names or recent. 'names' returns []models.WorkspaceName, 'recent' returns []models.Workspace unpaginated. Ignored if q is set."
// @Param limit query int false "Page size (default 50, max 200). Only applies to the default paginated view."
// @Param offset query int false "Page offset (default 0). Only applies to the default paginated view."
// @Success 200 {array} models.Workspace "Default paginated view, or view=recent"
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces [get]
func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	query := r.URL.Query()

	if q := strings.TrimSpace(query.Get("q")); q != "" {
		results, err := h.repo.Search(r.Context(), userID, q)
		if err != nil {
			serverError(w, r, err)
			return
		}
		utils.RespondJSON(w, http.StatusOK, results)
		return
	}

	switch query.Get("view") {
	case "names":
		names, err := h.repo.ListNames(r.Context(), userID)
		if err != nil {
			serverError(w, r, err)
			return
		}
		utils.RespondJSON(w, http.StatusOK, names)
		return
	case "recent":
		workspaces, err := h.repo.ListRecent(r.Context(), userID)
		if err != nil {
			serverError(w, r, err)
			return
		}
		utils.RespondJSON(w, http.StatusOK, workspaces)
		return
	}

	limit, offset, ok := parsePagination(w, r)
	if !ok {
		return
	}

	workspaces, err := h.repo.List(r.Context(), userID, limit, offset)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, workspaces)
}

func (h *WorkspaceHandler) recentOrNil(ctx context.Context, userID uuid.UUID) any {
	recent, err := h.repo.ListRecent(ctx, userID)
	if err != nil {
		logging.Logger.Error("failed to load recent workspaces for broadcast", slog.Any("error", err))
		return nil
	}
	return recent
}

// Create godoc
// @Summary Create a workspace
// @Tags workspace
// @Accept json
// @Produce json
// @Param body body createWorkspaceBody true "Workspace to create"
// @Success 201 {object} utils.CreateResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces [post]
func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	body, ok := utils.DecodeAndValidate[createWorkspaceBody]("WorkspaceHandler.Create", w, r)
	if !ok {
		return
	}

	workspace, err := h.repo.Create(r.Context(), body.Name, body.Description, userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	h.hub.Broadcast(userID, ws.Event{
		Type:        ws.WorkspaceCreated,
		WorkspaceID: &workspace.ID,
		Data:        workspace,
		Recent:      h.recentOrNil(r.Context(), userID),
	})
	utils.RespondCreated(w, &workspace.ID)
}

// Get godoc
// @Summary Get a workspace
// @Description Get a workspace's full detail (columns and tasks included).
// @Tags workspace
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 200 {object} models.WorkspaceDetail
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id} [get]
func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	detail, err := h.repo.GetByID(r.Context(), workspaceID, userID)
	if handleRepoError(w, r, err, "workspace not found") {
		return
	}
	h.resolveAssigneeAvatars(&detail)

	utils.RespondJSON(w, http.StatusOK, detail)
}

// Update godoc
// @Summary Update a workspace
// @Tags workspace
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param body body updateWorkspaceBody true "Fields to update"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id} [patch]
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
	if handleRepoError(w, r, err, "workspace not found") {
		return
	}

	h.hub.Broadcast(userID, ws.Event{Type: ws.WorkspaceUpdated, WorkspaceID: &workspace.ID, Data: workspace})
	utils.RespondUpdated(w)
}

// Delete godoc
// @Summary Delete a workspace
// @Description Soft-deletes a workspace.
// @Tags workspace
// @Param id path string true "Workspace ID"
// @Success 204 "No Content"
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id} [delete]
func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	workspaceID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	workspace, err := h.repo.SoftDelete(r.Context(), workspaceID, userID)
	if handleRepoError(w, r, err, "workspace not found") {
		return
	}

	h.hub.Broadcast(userID, ws.Event{
		Type:        ws.WorkspaceDeleted,
		WorkspaceID: &workspace.ID,
	})
	utils.RespondDeleted(w)
}

package handler

import (
	"kanbano-api/internal/models"
	"kanbano-api/internal/permissions"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/utils"
	"kanbano-api/internal/ws"
	"net/http"

	"github.com/google/uuid"
)

type ColumnHandler struct {
	repo *repository.ColumnRepository
	role *repository.RoleRepository
	hub  *ws.Hub
}

type createColumnBody struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type updateColumnBody struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Position *int    `json:"position,omitempty" validate:"omitempty,min=0,max=2147483647"`
}

func NewColumnHandler(repo *repository.ColumnRepository, role *repository.RoleRepository, hub *ws.Hub) *ColumnHandler {
	return &ColumnHandler{repo: repo, role: role, hub: hub}
}

// Names godoc
// @Summary List column names
// @Tags column
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 200 {array} models.ColumnName
// @Failure 400 {object} ErrorResponse "invalid id"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "guest access is limited to shared tasks"
// @Failure 404 {object} ErrorResponse "workspace not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/names [get]
func (h *ColumnHandler) Names(w http.ResponseWriter, r *http.Request) {
	_, workspaceID, access, ok := requireWorkspace(w, r, h.role)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanViewWorkspace(access), errGuestLimitedToTasks) {
		return
	}

	names, err := h.repo.ListNames(r.Context(), workspaceID)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, names)
}

// parseColumnContext resolves the workspace like requireWorkspace, then
// parses the {columnId} param. Columns have no access scope of their own:
// the workspace roles apply.
func (h *ColumnHandler) parseColumnContext(w http.ResponseWriter, r *http.Request) (userID, workspaceID, columnID uuid.UUID, access permissions.Access, ok bool) {
	userID, workspaceID, access, ok = requireWorkspace(w, r, h.role)
	if !ok {
		return userID, workspaceID, columnID, access, ok
	}

	columnID, ok = parseUUIDParam(w, r, "columnId")
	return userID, workspaceID, columnID, access, ok
}

func (h *ColumnHandler) broadcastColumn(userID, workspaceID uuid.UUID, eventType ws.EventType, column models.Column) {
	h.hub.Broadcast(userID, ws.Event{Type: eventType, WorkspaceID: &workspaceID, Data: column})
}

func (h *ColumnHandler) applyColumnChange(w http.ResponseWriter, r *http.Request, userID, workspaceID uuid.UUID, eventType ws.EventType, column models.Column, err error) bool {
	if handleRepoError(w, r, err, "column not found") {
		return false
	}

	h.broadcastColumn(userID, workspaceID, eventType, column)
	return true
}

// Create godoc
// @Summary Create a column
// @Tags column
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param body body createColumnBody true "Column to create"
// @Success 201 {object} utils.CreateResponse
// @Failure 400 {object} BadRequestResponse "invalid id | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required"
// @Failure 404 {object} ErrorResponse "workspace not found"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns [post]
func (h *ColumnHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, access, ok := requireWorkspace(w, r, h.role)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanCreateOrDeleteContent(access), errOwnerOrAdminRequired) {
		return
	}

	body, ok := utils.DecodeAndValidate[createColumnBody]("ColumnHandler.Create", w, r)
	if !ok {
		return
	}

	column, err := h.repo.Create(r.Context(), body.Name, workspaceID, userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	h.broadcastColumn(userID, workspaceID, ws.ColumnCreated, column)
	utils.RespondCreated(w, &column.ID)
}

// Update godoc
// @Summary Update or reorder a column
// @Tags column
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Param body body updateColumnBody true "Fields to update"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} BadRequestResponse "invalid id | invalid columnId | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "edit access required"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId} [patch]
func (h *ColumnHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, access, ok := h.parseColumnContext(w, r)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanEditContent(access), errEditAccessRequired) {
		return
	}

	body, ok := utils.DecodeAndValidate[updateColumnBody]("ColumnHandler.Update", w, r)
	if !ok {
		return
	}

	column, err := h.repo.Update(r.Context(), columnID, workspaceID, body.Name, userID)
	if handleRepoError(w, r, err, "column not found") {
		return
	}

	if body.Position != nil {
		column, err = h.repo.Reorder(r.Context(), columnID, workspaceID, *body.Position, userID)
		if handleRepoError(w, r, err, "column not found") {
			return
		}
	}

	h.broadcastColumn(userID, workspaceID, ws.ColumnUpdated, column)
	utils.RespondUpdated(w)
}

// Delete godoc
// @Summary Delete a column
// @Description Soft-deletes a column.
// @Tags column
// @Param id path string true "Workspace ID"
// @Param columnId path string true "Column ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "invalid id | invalid columnId"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required"
// @Failure 404 {object} ErrorResponse "workspace not found | column not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/columns/{columnId} [delete]
func (h *ColumnHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, access, ok := h.parseColumnContext(w, r)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanCreateOrDeleteContent(access), errOwnerOrAdminRequired) {
		return
	}

	column, err := h.repo.SoftDelete(r.Context(), columnID, workspaceID, userID)
	if h.applyColumnChange(w, r, userID, workspaceID, ws.ColumnDeleted, column, err) {
		utils.RespondDeleted(w)
	}
}

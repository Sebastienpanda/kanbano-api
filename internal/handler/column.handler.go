package handler

import (
	"errors"
	"kanbano-api/internal/models"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/utils"
	"kanbano-api/internal/ws"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ColumnHandler struct {
	repo          *repository.ColumnRepository
	workspaceRepo *repository.WorkspaceRepository
	hub           *ws.Hub
}

type createColumnBody struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type updateColumnBody struct {
	Name *string `json:"name" validate:"omitempty,min=1,max=100"`
}

type reorderColumnBody struct {
	Position int `json:"position" validate:"min=0"`
}

func NewColumnHandler(repo *repository.ColumnRepository, workspaceRepo *repository.WorkspaceRepository, hub *ws.Hub) *ColumnHandler {
	return &ColumnHandler{repo: repo, workspaceRepo: workspaceRepo, hub: hub}
}

func (h *ColumnHandler) NamesByWorkspaceName(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	name := r.URL.Query().Get("name")
	if name == "" {
		badRequest(w, "missing name parameter")
		return
	}

	workspaceID, err := h.workspaceRepo.GetIDByName(r.Context(), name, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			notFound(w, "workspace not found")
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

	utils.RespondJSON(w, http.StatusOK, names)
}

func (h *ColumnHandler) parseWorkspaceContext(w http.ResponseWriter, r *http.Request) (userID, workspaceID uuid.UUID, ok bool) {
	return requireWorkspace(w, r, h.workspaceRepo)
}

func (h *ColumnHandler) parseColumnContext(w http.ResponseWriter, r *http.Request) (userID, workspaceID, columnID uuid.UUID, ok bool) {
	userID, workspaceID, ok = h.parseWorkspaceContext(w, r)
	if !ok {
		return
	}

	columnID, ok = parseUUIDParam(w, r, "columnId")
	return
}

func (h *ColumnHandler) broadcastColumn(userID, workspaceID uuid.UUID, eventType ws.EventType, column models.Column) {
	h.hub.Broadcast(userID, ws.Event{Type: eventType, WorkspaceID: &workspaceID, Data: column})
}

// applyColumnChange handles the common tail of Update/Reorder/Delete: turn a repo
// error into a 404/500, otherwise broadcast the change (WS carries the payload).
// Returns true when the change succeeded and the caller should send its ack.
func (h *ColumnHandler) applyColumnChange(w http.ResponseWriter, userID, workspaceID uuid.UUID, eventType ws.EventType, column models.Column, err error) bool {
	if handleRepoError(w, err, "column not found") {
		return false
	}

	h.broadcastColumn(userID, workspaceID, eventType, column)
	return true
}

func (h *ColumnHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, ok := h.parseWorkspaceContext(w, r)
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[createColumnBody]("ColumnHandler.Create", w, r)
	if !ok {
		return
	}

	column, err := h.repo.Create(r.Context(), body.Name, workspaceID)
	if err != nil {
		serverError(w, err)
		return
	}
	h.broadcastColumn(userID, workspaceID, ws.ColumnCreated, column)
	utils.RespondCreated(w, &column.ID)
}

func (h *ColumnHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseColumnContext(w, r)
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[updateColumnBody]("ColumnHandler.Update", w, r)
	if !ok {
		return
	}

	column, err := h.repo.Update(r.Context(), columnID, workspaceID, body.Name)
	if h.applyColumnChange(w, userID, workspaceID, ws.ColumnUpdated, column, err) {
		utils.RespondUpdated(w)
	}
}

func (h *ColumnHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseColumnContext(w, r)
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[reorderColumnBody]("ColumnHandler.Reorder", w, r)
	if !ok {
		return
	}

	column, err := h.repo.Reorder(r.Context(), columnID, workspaceID, body.Position)
	if h.applyColumnChange(w, userID, workspaceID, ws.ColumnUpdated, column, err) {
		utils.RespondUpdated(w)
	}
}

func (h *ColumnHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, columnID, ok := h.parseColumnContext(w, r)
	if !ok {
		return
	}

	column, err := h.repo.SoftDelete(r.Context(), columnID, workspaceID)
	if h.applyColumnChange(w, userID, workspaceID, ws.ColumnDeleted, column, err) {
		utils.RespondDeleted(w)
	}
}

package handler

import (
	"context"
	"errors"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/models"
	"kanbano-api/internal/permissions"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/storage"
	"kanbano-api/internal/utils"
	"kanbano-api/internal/ws"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WorkspaceHandler struct {
	repo         *repository.WorkspaceRepository
	role         *repository.RoleRepository
	organisation *repository.OrganisationRepository
	store        *storage.Client
	hub          *ws.Hub
}

type createWorkspaceBody struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`
}

type updateWorkspaceBody struct {
	Name        *string                     `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string                     `json:"description,omitempty" validate:"omitempty,max=2000"`
	Members     []updateWorkspaceMemberBody `json:"members,omitempty" validate:"omitempty,unique=ID,dive"`
}

type updateWorkspaceMemberBody struct {
	ID         uuid.UUID `json:"id" validate:"required"`
	Visibility *string   `json:"visibility,omitempty" validate:"omitempty,oneof=private public"`
	Role       *string   `json:"role,omitempty" validate:"omitempty,oneof=view edit"`
}

func NewWorkspaceHandler(repo *repository.WorkspaceRepository, role *repository.RoleRepository, organisation *repository.OrganisationRepository, store *storage.Client, hub *ws.Hub) *WorkspaceHandler {
	return &WorkspaceHandler{repo: repo, role: role, organisation: organisation, store: store, hub: hub}
}

func (h *WorkspaceHandler) resolveAssigneeAvatars(detail *models.WorkspaceDetail) {
	for m := range detail.Members {
		detail.Members[m].Avatar = avatarSet(h.store, detail.Members[m].ID, detail.Members[m].AvatarVersion)
	}
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
// @Param view query string false "'names' returns []models.WorkspaceName, 'recent' returns []models.Workspace unpaginated. Ignored if q is set." Enums(names, recent)
// @Param limit query int false "Page size (default 50, max 200). Only applies to the default paginated view."
// @Param offset query int false "Page offset (default 0). Only applies to the default paginated view."
// Swagger 2 has no oneOf: swag keeps only the last 200, the two lines before
// it just publish the schemas of the q and view=names variants.
// @Success 200 {array} models.WorkspaceSearchResult "q set"
// @Success 200 {array} models.WorkspaceName "view=names"
// @Success 200 {array} models.Workspace "Default paginated view, or view=recent"
// @Failure 400 {object} ErrorResponse "invalid limit | invalid offset | invalid q"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces [get]
func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	query := r.URL.Query()

	if q := strings.TrimSpace(query.Get("q")); q != "" {
		if strings.ContainsRune(q, 0) {
			badRequest(w, r, "invalid q")
			return
		}
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
// @Description Creates a private workspace in the organisation the caller owns. Only an organisation owner may create one (403 for a user who owns no organisation, e.g. after deleting it).
// @Tags workspace
// @Accept json
// @Produce json
// @Param body body createWorkspaceBody true "Workspace to create"
// @Success 201 {object} utils.CreateResponse
// @Failure 400 {object} BadRequestResponse "could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner required"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces [post]
func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	body, ok := utils.DecodeAndValidate[createWorkspaceBody]("WorkspaceHandler.Create", w, r)
	if !ok {
		return
	}

	organisationID, err := h.organisation.OwnedID(r.Context(), userID)
	orgRole := permissions.Owner
	if errors.Is(err, pgx.ErrNoRows) {
		orgRole = ""
	} else if err != nil {
		serverError(w, r, err)
		return
	}
	if !requirePermission(w, r, permissions.CanCreateWorkspace(orgRole), errOwnerRequired) {
		return
	}

	workspace, err := h.repo.Create(r.Context(), organisationID, body.Name, body.Description, userID)
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
// @Description Get a workspace's full detail (columns and tasks included). A guest only gets the tasks shared with them (and their columns), and no members.
// @Tags workspace
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 200 {object} models.WorkspaceDetail
// @Failure 400 {object} ErrorResponse "invalid id"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 404 {object} ErrorResponse "workspace not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id} [get]
func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, access, ok := requireWorkspace(w, r, h.role)
	if !ok {
		return
	}

	detail, err := h.repo.GetByID(r.Context(), workspaceID, userID, !permissions.CanViewWorkspace(access))
	if handleRepoError(w, r, err, "workspace not found") {
		return
	}
	h.resolveAssigneeAvatars(&detail)

	utils.RespondJSON(w, http.StatusOK, detail)
}

// Update godoc
// @Summary Update a workspace
// @Description Updates name/description and, in the same transaction, the access of each listed member on this workspace: visibility ('private' removes the member's access, 'public' grants it) and/or role ('view' or 'edit'). For each member, an omitted field keeps its current value, or its default ('private' / 'view') if the member had no access yet: to add a member, send visibility 'public'. Allowed for the organisation owner, or an admin the workspace is shared with (403 for other users who see the workspace, 404 "workspace not found" for the others). An admin only manages members, never another admin or the owner; nobody changes their own access (403). Every listed member must belong to the organisation (404 otherwise). Nothing is changed when a check fails.
// @Tags workspace
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param body body updateWorkspaceBody true "Fields to update"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} BadRequestResponse "invalid id | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required | cannot change your own access | cannot manage this member's access"
// @Failure 404 {object} ErrorResponse "workspace not found | member not found in this organisation"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id} [patch]
func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, access, ok := requireWorkspace(w, r, h.role)
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[updateWorkspaceBody]("WorkspaceHandler.Update", w, r)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanEditWorkspace(access), errOwnerOrAdminRequired) {
		return
	}

	members := make([]repository.WorkspaceMemberAccess, len(body.Members))
	for i, m := range body.Members {
		members[i] = repository.WorkspaceMemberAccess{MemberID: m.ID, Visibility: m.Visibility, Role: m.Role}
	}
	if !requireMembersAccessChange(w, r, h.role, workspaceID, userID, access, members) {
		return
	}

	workspace, err := h.repo.Update(r.Context(), workspaceID, userID, body.Name, body.Description, members)
	if handleRepoError(w, r, err, "workspace not found") {
		return
	}

	h.hub.Broadcast(userID, ws.Event{Type: ws.WorkspaceUpdated, WorkspaceID: &workspace.ID, Data: workspace})
	utils.RespondUpdated(w)
}

// Delete godoc
// @Summary Delete a workspace
// @Description Soft-deletes a workspace, along with its columns and tasks. Only the organisation owner may call this (403 for other users who see the workspace, 404 "workspace not found" for the others).
// @Tags workspace
// @Param id path string true "Workspace ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "invalid id"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner required"
// @Failure 404 {object} ErrorResponse "workspace not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id} [delete]
func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, access, ok := requireWorkspace(w, r, h.role)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanDeleteWorkspace(access), errOwnerRequired) {
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

// requireMembersAccessChange checks that the caller may change the access
// of every listed member: each one belongs to the workspace's organisation
// (404 otherwise), is not the caller, and is a user the caller may manage
// (permissions.CanManageWorkspaceAccess), 403 otherwise.
func requireMembersAccessChange(w http.ResponseWriter, r *http.Request, roleRepo *repository.RoleRepository, workspaceID, userID uuid.UUID, access permissions.Access, members []repository.WorkspaceMemberAccess) bool {
	if len(members) == 0 {
		return true
	}

	memberIDs := make([]uuid.UUID, len(members))
	for i, m := range members {
		memberIDs[i] = m.MemberID
	}
	orgRoles, err := roleRepo.OrgRoles(r.Context(), workspaceID, memberIDs)
	if err != nil {
		serverError(w, r, err)
		return false
	}

	for _, m := range members {
		orgRole, found := orgRoles[m.MemberID]
		switch {
		case !found:
			notFound(w, r, "member not found in this organisation")
			return false
		case m.MemberID == userID:
			forbidden(w, r, "cannot change your own access")
			return false
		case !permissions.CanManageWorkspaceAccess(access, orgRole):
			forbidden(w, r, "cannot manage this member's access")
			return false
		}
	}
	return true
}

package handler

import (
	"kanbano-api/internal/permissions"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/utils"
	"net/http"

	"github.com/google/uuid"
)

type setWorkspaceRoleBody struct {
	Visibility *string `json:"visibility,omitempty" validate:"omitempty,oneof=private public"`
	Role       *string `json:"role,omitempty" validate:"omitempty,oneof=view edit"`
}

type RoleHandler struct {
	role         *repository.RoleRepository
	organisation *repository.OrganisationRepository
	workspace    *repository.WorkspaceRepository
}

func NewRoleHandler(role *repository.RoleRepository, organisation *repository.OrganisationRepository, workspace *repository.WorkspaceRepository) *RoleHandler {
	return &RoleHandler{role: role, organisation: organisation, workspace: workspace}
}

type roleResponse struct {
	Role string `json:"role" enums:"owner,admin,member,view,edit"`
}

// Role godoc
// @Summary Get the caller's effective role for a scope
// @Description Returns the caller's effective role at a given point in time. With organisation_id set, returns their role in that organisation ('owner', 'admin' or 'member'; 404 if they do not belong to it). With workspace_id set, returns their workspace role: 'edit' if they may modify columns and tasks, 'view' otherwise (always 'view' for a guest). With task_id as well, returns their role on that task: the workspace role for a member, the invitation role for a guest ('edit' then only allows changing the description).
// @Tags role
// @Produce json
// @Param organisation_id query string false "Organisation ID (ignored when workspace_id is set)"
// @Param workspace_id query string false "Workspace ID"
// @Param task_id query string false "Task ID (requires workspace_id)"
// @Success 200 {object} roleResponse
// @Failure 400 {object} ErrorResponse "organisation_id or workspace_id required | invalid organisation_id | invalid workspace_id | invalid task_id"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "no access to this workspace | no access to this task"
// @Failure 404 {object} ErrorResponse "organisation not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /roles [get]
func (h *RoleHandler) Role(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	query := r.URL.Query()

	workspaceIDStr := query.Get("workspace_id")
	if workspaceIDStr == "" {
		h.organisationRole(w, r, userID, query.Get("organisation_id"))
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		badRequest(w, r, "invalid workspace_id")
		return
	}

	taskID := uuid.Nil
	if taskIDStr := query.Get("task_id"); taskIDStr != "" {
		taskID, err = uuid.Parse(taskIDStr)
		if err != nil {
			badRequest(w, r, "invalid task_id")
			return
		}
	}

	access, err := h.role.ResolveAccess(r.Context(), workspaceID, taskID, userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if !permissions.CanSeeWorkspace(access) {
		forbidden(w, r, "no access to this workspace")
		return
	}

	if taskID != uuid.Nil {
		inWorkspace, err := h.role.TaskInWorkspace(r.Context(), taskID, workspaceID)
		if err != nil {
			serverError(w, r, err)
			return
		}
		if !inWorkspace {
			forbidden(w, r, "no access to this task")
			return
		}
	}

	role := permissions.View
	switch {
	case taskID != uuid.Nil && !permissions.CanViewTask(access):
		forbidden(w, r, "no access to this task")
		return
	case permissions.CanEditContent(access), taskID != uuid.Nil && access.GuestRole == permissions.Edit:
		role = permissions.Edit
	}

	utils.RespondJSON(w, http.StatusOK, roleResponse{Role: role})
}

// organisationRole answers GET /roles?organisation_id=...
func (h *RoleHandler) organisationRole(w http.ResponseWriter, r *http.Request, userID uuid.UUID, organisationIDStr string) {
	if organisationIDStr == "" {
		badRequest(w, r, "organisation_id or workspace_id required")
		return
	}
	organisationID, err := uuid.Parse(organisationIDStr)
	if err != nil {
		badRequest(w, r, "invalid organisation_id")
		return
	}

	role, err := h.organisation.GetRole(r.Context(), organisationID, userID)
	if handleRepoError(w, r, err, "organisation not found") {
		return
	}
	utils.RespondJSON(w, http.StatusOK, roleResponse{Role: role})
}

// Abilities godoc
// @Summary Get the caller's CASL abilities for a workspace
// @Description Returns the caller's permissions on the workspace as a list of CASL rules (action/subject/fields/conditions), consumable directly by @casl/ability's createMongoAbility(rules) on the Angular side. Actions: read, create, update, delete on Workspace, Column and Task, plus share (manage the members' access) on Workspace, and assign and invite (guests) on Task. A guest gets read on the Workspace and on each task shared with them, and update restricted to the description field on the tasks they were invited to with 'edit'.
// @Tags role
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 200 {array} models.AbilityRule
// @Failure 400 {object} ErrorResponse "invalid id"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 404 {object} ErrorResponse "workspace not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/abilities [get]
func (h *RoleHandler) Abilities(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, access, ok := requireWorkspace(w, r, h.role)
	if !ok {
		return
	}

	rules, err := h.role.BuildWorkspaceAbilities(r.Context(), workspaceID, userID, access)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, rules)
}

// SetWorkspaceRole godoc
// @Summary Override a member's visibility and/or role on a workspace
// @Description Sets whether this workspace is visible to the member ('private', the default, or 'public') and/or their role on it ('view', the default, or 'edit'). Both fields are optional and independent; omitting one leaves it unchanged (or at its default if this is the member's first override on this workspace). Allowed for the organisation owner, or an admin the workspace is shared with; an admin only manages members, never another admin or the owner, and nobody changes their own access. The member must belong to the organisation. Same effect as PATCH /workspaces/{id} with a single entry in members.
// @Tags role
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param memberID path string true "Member ID (user ID)"
// @Param body body setWorkspaceRoleBody true "New visibility and/or role"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} BadRequestResponse "invalid id | invalid memberID | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required | cannot change your own access | cannot manage this member's access"
// @Failure 404 {object} ErrorResponse "workspace not found | member not found in this organisation"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/members/{memberID}/role [put]
func (h *RoleHandler) SetWorkspaceRole(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, access, ok := requireWorkspace(w, r, h.role)
	if !ok {
		return
	}

	memberID, ok := parseUUIDParam(w, r, "memberID")
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[setWorkspaceRoleBody]("RoleHandler.SetWorkspaceRole", w, r)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanEditWorkspace(access), errOwnerOrAdminRequired) {
		return
	}

	members := []repository.WorkspaceMemberAccess{{MemberID: memberID, Visibility: body.Visibility, Role: body.Role}}
	if !requireMembersAccessChange(w, r, h.role, workspaceID, userID, access, members) {
		return
	}

	_, err := h.workspace.Update(r.Context(), workspaceID, userID, nil, nil, members)
	if handleRepoError(w, r, err, "workspace not found") {
		return
	}

	utils.RespondUpdated(w)
}

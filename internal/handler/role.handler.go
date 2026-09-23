package handler

import (
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
	Role string `json:"role"`
}

// Role godoc
// @Summary Get the caller's effective role for a scope
// @Description Returns the caller's effective role ('edit' or 'view') at a given point in time. With no query params, returns their organisation role. With workspace_id set, returns their workspace role, optionally narrowed with task_id to the role that actually applies there — a task-specific assignment always wins over the organisation role.
// @Tags role
// @Produce json
// @Param workspace_id query string false "Workspace ID"
// @Param task_id query string false "Task ID (requires workspace_id)"
// @Success 200 {object} roleResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /roles [get]
func (h *RoleHandler) Role(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)
	query := r.URL.Query()

	workspaceIDStr := query.Get("workspace_id")
	if workspaceIDStr == "" {
		role, err := h.organisation.GetMemberRole(r.Context(), userID)
		if handleRepoError(w, r, err, "no organisation membership found") {
			return
		}
		utils.RespondJSON(w, http.StatusOK, roleResponse{Role: role})
		return
	}

	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		badRequest(w, r, "invalid workspace_id")
		return
	}

	taskIDStr := query.Get("task_id")
	if taskIDStr == "" {
		hasAccess, err := h.role.HasWorkspaceAccess(r.Context(), workspaceID, userID)
		if err != nil {
			serverError(w, r, err)
			return
		}
		if !hasAccess {
			forbidden(w, r, "no access to this workspace")
			return
		}
		hasEditAccess, err := h.role.HasWorkspaceEditAccess(r.Context(), workspaceID, userID)
		if err != nil {
			serverError(w, r, err)
			return
		}
		role := "view"
		if hasEditAccess {
			role = "edit"
		}
		utils.RespondJSON(w, http.StatusOK, roleResponse{Role: role})
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		badRequest(w, r, "invalid task_id")
		return
	}

	role, hasAccess, err := h.role.ResolveTaskRole(r.Context(), workspaceID, taskID, userID)
	if handleRepoError(w, r, err, "workspace not found") {
		return
	}
	if !hasAccess {
		forbidden(w, r, "no access to this task")
		return
	}

	utils.RespondJSON(w, http.StatusOK, roleResponse{Role: role})
}

// Abilities godoc
// @Summary Get the caller's CASL abilities for a workspace
// @Description Returns the caller's permissions on the workspace as a list of CASL rules (action/subject/conditions), consumable directly by @casl/ability's createMongoAbility(rules) on the Angular side. Covers the Workspace, Column and Task subjects, including per-task overrides from task assignments.
// @Tags role
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 200 {array} models.AbilityRule
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/abilities [get]
func (h *RoleHandler) Abilities(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, ok := requireWorkspace(w, r, h.workspace)
	if !ok {
		return
	}

	rules, hasAccess, err := h.role.BuildWorkspaceAbilities(r.Context(), workspaceID, userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if !hasAccess {
		forbidden(w, r, "no access to this workspace")
		return
	}

	utils.RespondJSON(w, http.StatusOK, rules)
}

// SetWorkspaceRole godoc
// @Summary Override a member's visibility and/or role on a workspace
// @Description Sets whether this workspace is visible to the member ('private', the default, or 'public') and/or their role on it ('view', the default, or 'edit'). Both fields are optional and independent; omitting one leaves it unchanged (or at its default if this is the member's first override on this workspace). Only callers with organisation-level 'edit' (manage) may call this.
// @Tags role
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param memberID path string true "Member ID (user ID)"
// @Param body body setWorkspaceRoleBody true "New visibility and/or role"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /workspaces/{id}/members/{memberID}/role [put]
func (h *RoleHandler) SetWorkspaceRole(w http.ResponseWriter, r *http.Request) {
	userID, workspaceID, ok := requireWorkspace(w, r, h.workspace)
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

	orgRole, err := h.organisation.GetMemberRole(r.Context(), userID)
	if handleRepoError(w, r, err, "no organisation membership found") {
		return
	}
	if orgRole != "edit" {
		forbidden(w, r, "organisation-level manage access required")
		return
	}

	if err := h.role.SetWorkspaceMemberAccess(r.Context(), workspaceID, memberID, body.Visibility, body.Role); err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondUpdated(w)
}

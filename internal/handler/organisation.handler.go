package handler

import (
	"context"
	"errors"
	"kanbano-api/internal/brevo"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/models"
	"kanbano-api/internal/permissions"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/storage"
	"kanbano-api/internal/utils"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type OrganisationHandler struct {
	repo            *repository.OrganisationRepository
	userRepo        *repository.UserRepository
	store           *storage.Client
	mailer          *brevo.Client
	invitationTplID int
	frontendBaseURL string
}

type OrganisationHandlerConfig struct {
	Repo            *repository.OrganisationRepository
	UserRepo        *repository.UserRepository
	Store           *storage.Client
	Mailer          *brevo.Client
	InvitationTplID int
	FrontendBaseURL string
}

func NewOrganisationHandler(cfg OrganisationHandlerConfig) *OrganisationHandler {
	return &OrganisationHandler{
		repo:            cfg.Repo,
		userRepo:        cfg.UserRepo,
		store:           cfg.Store,
		mailer:          cfg.Mailer,
		invitationTplID: cfg.InvitationTplID,
		frontendBaseURL: cfg.FrontendBaseURL,
	}
}

type inviteBody struct {
	Email string  `json:"email" validate:"required,email"`
	Role  *string `json:"role,omitempty" validate:"omitempty,oneof=admin member"`
}

type updateInvitationBody struct {
	Status string `json:"status" validate:"required,oneof=accepted declined"`
}

type updateOrganisationBody struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

type setMemberRoleBody struct {
	Role string `json:"role" validate:"required,oneof=admin member"`
}

type organisationResponse struct {
	ID      uuid.UUID        `json:"id"`
	Name    string           `json:"name"`
	UserID  uuid.UUID        `json:"user_id"`
	Role    string           `json:"role" enums:"owner,admin,member"`
	Members []memberResponse `json:"members"`
}

type memberResponse struct {
	ID       uuid.UUID         `json:"id"`
	Name     string            `json:"name"`
	Avatar   *models.AvatarSet `json:"avatar" extensions:"x-nullable"`
	JoinedAt *time.Time        `json:"joined_at" extensions:"x-nullable"`
	Role     string            `json:"role" enums:"owner,admin,member"`
}

// requireOrganisation parses the {orgId} param and resolves the caller's
// role in that organisation. Writes a 404 and returns false if the
// organisation does not exist for the caller.
func (h *OrganisationHandler) requireOrganisation(w http.ResponseWriter, r *http.Request) (userID, organisationID uuid.UUID, role string, ok bool) {
	userID = userIDFromContext(r)

	organisationID, ok = parseUUIDParam(w, r, "orgId")
	if !ok {
		return userID, organisationID, "", false
	}

	role, err := h.repo.GetRole(r.Context(), organisationID, userID)
	if handleRepoError(w, r, err, "organisation not found") {
		return userID, organisationID, "", false
	}
	return userID, organisationID, role, true
}

// List godoc
// @Summary List the caller's organisations
// @Description Returns the organisations the caller owns (first, at most one) or belongs to as admin or member, with their role in each.
// @Tags organisation
// @Produce json
// @Success 200 {array} models.OrganisationSummary
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisations [get]
func (h *OrganisationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	organisations, err := h.repo.ListForUser(r.Context(), userID)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, organisations)
}

// Get godoc
// @Summary Get an organisation
// @Description Returns the organisation, the caller's role in it and its members, the owner first. Visible to every person of the organisation.
// @Tags organisation
// @Produce json
// @Param orgId path string true "Organisation ID"
// @Success 200 {object} organisationResponse
// @Failure 400 {object} ErrorResponse "invalid orgId"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 404 {object} ErrorResponse "organisation not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisations/{orgId} [get]
func (h *OrganisationHandler) Get(w http.ResponseWriter, r *http.Request) {
	_, organisationID, role, ok := h.requireOrganisation(w, r)
	if !ok {
		return
	}

	organisation, err := h.repo.Get(r.Context(), organisationID)
	if handleRepoError(w, r, err, "organisation not found") {
		return
	}

	utils.RespondJSON(w, http.StatusOK, h.response(organisation, role))
}

func (h *OrganisationHandler) response(org models.Organisation, role string) organisationResponse {
	members := make([]memberResponse, len(org.Members))
	for i, m := range org.Members {
		name := "Anonyme"
		if m.Name != nil && *m.Name != "" {
			name = *m.Name
		}
		members[i] = memberResponse{
			ID:       m.ID,
			Name:     name,
			Avatar:   avatarSet(h.store, m.ID, m.AvatarVersion),
			JoinedAt: m.JoinedAt,
			Role:     m.Role,
		}
	}
	return organisationResponse{ID: org.ID, Name: org.Name, UserID: org.UserID, Role: role, Members: members}
}

// Update godoc
// @Summary Rename an organisation
// @Description Allowed for the organisation owner only.
// @Tags organisation
// @Accept json
// @Produce json
// @Param orgId path string true "Organisation ID"
// @Param body body updateOrganisationBody true "New name"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} BadRequestResponse "invalid orgId | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner required"
// @Failure 404 {object} ErrorResponse "organisation not found"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisations/{orgId} [patch]
func (h *OrganisationHandler) Update(w http.ResponseWriter, r *http.Request) {
	_, organisationID, role, ok := h.requireOrganisation(w, r)
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[updateOrganisationBody]("OrganisationHandler.Update", w, r)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanManageOrg(role), errOwnerRequired) {
		return
	}

	err := h.repo.Rename(r.Context(), organisationID, body.Name)
	if handleRepoError(w, r, err, "organisation not found") {
		return
	}

	utils.RespondUpdated(w)
}

// Delete godoc
// @Summary Delete an organisation
// @Description Soft-deletes the organisation with all its workspaces, columns and tasks: nobody sees them anymore. The owner does not get a new organisation, so they can no longer create workspaces. Allowed for the organisation owner only.
// @Tags organisation
// @Param orgId path string true "Organisation ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "invalid orgId"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner required"
// @Failure 404 {object} ErrorResponse "organisation not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisations/{orgId} [delete]
func (h *OrganisationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, organisationID, role, ok := h.requireOrganisation(w, r)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanManageOrg(role), errOwnerRequired) {
		return
	}

	err := h.repo.SoftDelete(r.Context(), organisationID, userID)
	if handleRepoError(w, r, err, "organisation not found") {
		return
	}

	utils.RespondDeleted(w)
}

// SetMemberRole godoc
// @Summary Change a member's organisation role
// @Description Switches a member between 'admin' and 'member'. Allowed for the organisation owner only, who cannot change their own role.
// @Tags organisation
// @Accept json
// @Produce json
// @Param orgId path string true "Organisation ID"
// @Param memberId path string true "Member ID (user ID)"
// @Param body body setMemberRoleBody true "New role"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} BadRequestResponse "invalid orgId | invalid memberId | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner required | cannot change your own role"
// @Failure 404 {object} ErrorResponse "organisation not found | member not found"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisations/{orgId}/members/{memberId} [patch]
func (h *OrganisationHandler) SetMemberRole(w http.ResponseWriter, r *http.Request) {
	userID, organisationID, role, ok := h.requireOrganisation(w, r)
	if !ok {
		return
	}

	memberID, ok := parseUUIDParam(w, r, "memberId")
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[setMemberRoleBody]("OrganisationHandler.SetMemberRole", w, r)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanManageOrg(role), errOwnerRequired) ||
		!requirePermission(w, r, memberID != userID, "cannot change your own role") {
		return
	}

	err := h.repo.SetMemberRole(r.Context(), organisationID, memberID, body.Role)
	if handleRepoError(w, r, err, "member not found") {
		return
	}

	utils.RespondUpdated(w)
}

// RemoveMember godoc
// @Summary Remove a member from an organisation
// @Description Removes an admin or member from the organisation, along with their access to its workspaces and their assignments on its tasks. Allowed for the organisation owner only, who cannot remove themselves.
// @Tags organisation
// @Param orgId path string true "Organisation ID"
// @Param memberId path string true "Member ID (user ID)"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "invalid orgId | invalid memberId"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner required | cannot remove yourself"
// @Failure 404 {object} ErrorResponse "organisation not found | member not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisations/{orgId}/members/{memberId} [delete]
func (h *OrganisationHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userID, organisationID, role, ok := h.requireOrganisation(w, r)
	if !ok {
		return
	}

	memberID, ok := parseUUIDParam(w, r, "memberId")
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanManageOrg(role), errOwnerRequired) ||
		!requirePermission(w, r, memberID != userID, "cannot remove yourself") {
		return
	}

	err := h.repo.RemoveMember(r.Context(), organisationID, memberID)
	if handleRepoError(w, r, err, "member not found") {
		return
	}

	utils.RespondDeleted(w)
}

type memberProfileResponse struct {
	ID         uuid.UUID              `json:"id"`
	Email      string                 `json:"email"`
	Name       string                 `json:"name"`
	Avatar     *models.AvatarSet      `json:"avatar" extensions:"x-nullable"`
	Role       string                 `json:"role" enums:"owner,admin,member"`
	Workspaces []models.WorkspaceRole `json:"workspaces"`
}

// MemberProfile godoc
// @Summary Get a member's profile within an organisation
// @Description Returns the person's identity, organisation role ('owner', 'admin' or 'member') and, for each workspace, whether it is shared with them (visibility 'public', 'private' otherwise) and their role there ('view' by default, 'edit'). The owner gets every workspace of the organisation, an admin only the workspaces shared with them. Allowed for the owner and admins.
// @Tags organisation
// @Produce json
// @Param orgId path string true "Organisation ID"
// @Param memberId path string true "Member ID (user ID)"
// @Success 200 {object} memberProfileResponse
// @Failure 400 {object} ErrorResponse "invalid orgId | invalid memberId"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner or admin required"
// @Failure 404 {object} ErrorResponse "organisation not found | member not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisations/{orgId}/members/{memberId}/profile [get]
func (h *OrganisationHandler) MemberProfile(w http.ResponseWriter, r *http.Request) {
	userID, organisationID, role, ok := h.requireOrganisation(w, r)
	if !ok {
		return
	}

	memberID, ok := parseUUIDParam(w, r, "memberId")
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanViewMemberAccess(role), errOwnerOrAdminRequired) {
		return
	}

	profile, err := h.repo.GetMemberProfile(r.Context(), organisationID, userID, memberID)
	if handleRepoError(w, r, err, "member not found") {
		return
	}

	name := "Anonyme"
	if profile.Name != nil && *profile.Name != "" {
		name = *profile.Name
	}

	utils.RespondJSON(w, http.StatusOK, memberProfileResponse{
		ID:         profile.ID,
		Email:      profile.Email,
		Name:       name,
		Avatar:     avatarSet(h.store, profile.ID, profile.AvatarVersion),
		Role:       profile.Role,
		Workspaces: profile.Workspaces,
	})
}

// Invite godoc
// @Summary Invite someone into an organisation
// @Description Sends an organisation-level invitation by email. role is the organisation role granted on acceptance ('member' by default, or 'admin'). If the invitee is a guest on tasks of the organisation, accepting turns those guest accesses into access to the tasks' workspaces. Allowed for the organisation owner only.
// @Tags organisation
// @Accept json
// @Produce json
// @Param orgId path string true "Organisation ID"
// @Param body body inviteBody true "Invitation to create"
// @Success 201 {object} utils.CreateResponse
// @Failure 400 {object} BadRequestResponse "invalid orgId | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner required"
// @Failure 404 {object} ErrorResponse "organisation not found"
// @Failure 409 {object} ErrorResponse "an invitation is already pending for this email | this user already belongs to the organisation"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisations/{orgId}/invitations [post]
func (h *OrganisationHandler) Invite(w http.ResponseWriter, r *http.Request) {
	userID, organisationID, role, ok := h.requireOrganisation(w, r)
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[inviteBody]("OrganisationHandler.Invite", w, r)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanManageOrg(role), errOwnerRequired) {
		return
	}

	invitation, err := h.repo.CreateInvitation(r.Context(), repository.InvitationParams{
		OrganisationID: organisationID,
		Email:          body.Email,
		InvitedBy:      userID,
		Role:           body.Role,
	})
	switch {
	case errors.Is(err, repository.ErrInvitationAlreadyPending):
		conflict(w, r, "an invitation is already pending for this email")
		return
	case errors.Is(err, repository.ErrAlreadyOrganisationMember):
		conflict(w, r, "this user already belongs to the organisation")
		return
	case err != nil:
		serverError(w, r, err)
		return
	}

	inviter, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		logging.Logger.Error("failed to load inviter for invitation email", slog.Any("error", err))
	} else {
		h.sendInvitationEmail(r, invitation, inviter)
	}

	utils.RespondCreated(w, &invitation.ID)
}

func (h *OrganisationHandler) sendInvitationEmail(_ *http.Request, invitation models.OrganisationInvitation, inviter models.User) {
	if h.mailer == nil || h.invitationTplID == 0 {
		return
	}

	inviterName := inviter.Email
	if inviter.Name != nil && *inviter.Name != "" {
		inviterName = *inviter.Name
	}

	params := map[string]any{
		"INVITER_NAME":  inviterName,
		"INVITER_EMAIL": inviter.Email,
		"ACCEPT_URL":    h.frontendBaseURL + "/invitations/" + invitation.ID.String() + "?action=accept",
		"DECLINE_URL":   h.frontendBaseURL + "/invitations/" + invitation.ID.String() + "?action=decline",
	}

	safeGo(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.mailer.SendTemplateEmail(ctx, invitation.Email, h.invitationTplID, params); err != nil {
			logging.Logger.Error("failed to send invitation email", slog.Any("error", err))
		}
	})
}

// SentInvitations godoc
// @Summary List the pending invitations of an organisation
// @Description Allowed for the organisation owner only.
// @Tags organisation
// @Produce json
// @Param orgId path string true "Organisation ID"
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.OrganisationInvitation
// @Failure 400 {object} ErrorResponse "invalid orgId | invalid limit | invalid offset"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} ErrorResponse "organisation owner required"
// @Failure 404 {object} ErrorResponse "organisation not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisations/{orgId}/invitations [get]
func (h *OrganisationHandler) SentInvitations(w http.ResponseWriter, r *http.Request) {
	_, organisationID, role, ok := h.requireOrganisation(w, r)
	if !ok {
		return
	}

	limit, offset, ok := parsePagination(w, r)
	if !ok {
		return
	}

	if !requirePermission(w, r, permissions.CanManageOrg(role), errOwnerRequired) {
		return
	}

	invitations, err := h.repo.ListSentInvitations(r.Context(), organisationID, limit, offset)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, invitations)
}

// ReceivedInvitations godoc
// @Summary List organisation invitations received by the current user
// @Tags organisation
// @Produce json
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.OrganisationInvitation
// @Failure 400 {object} ErrorResponse "invalid limit | invalid offset"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 404 {object} ErrorResponse "user not found"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisation-invitations/received [get]
func (h *OrganisationHandler) ReceivedInvitations(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	limit, offset, ok := parsePagination(w, r)
	if !ok {
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if handleRepoError(w, r, err, "user not found") {
		return
	}

	invitations, err := h.repo.ListReceivedInvitations(r.Context(), user.Email, limit, offset)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, invitations)
}

// UpdateInvitationStatus godoc
// @Summary Accept or decline an organisation invitation
// @Description Accepting makes the caller an admin or member of the organisation. Their guest accesses on tasks of that organisation are removed and replaced by access to the tasks' workspaces, with the guest role ('edit' if they had it on at least one task of the workspace).
// @Tags organisation
// @Accept json
// @Produce json
// @Param id path string true "Invitation ID"
// @Param body body updateInvitationBody true "New status"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} BadRequestResponse "invalid id | could not read body | could not decode body | errors: {field: message} (validation)"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 404 {object} ErrorResponse "user not found | invitation not found"
// @Failure 415 {object} UnsupportedMediaTypeResponse
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /organisation-invitations/{id} [patch]
func (h *OrganisationHandler) UpdateInvitationStatus(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	invitationID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[updateInvitationBody]("OrganisationHandler.UpdateInvitationStatus", w, r)
	if !ok {
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if handleRepoError(w, r, err, "user not found") {
		return
	}

	if body.Status == "accepted" {
		_, err = h.repo.AcceptInvitation(r.Context(), invitationID, userID, user.Email)
	} else {
		_, err = h.repo.DeclineInvitation(r.Context(), invitationID, user.Email)
	}
	if handleRepoError(w, r, err, "invitation not found") {
		return
	}

	utils.RespondUpdated(w)
}

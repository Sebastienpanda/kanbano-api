package handler

import (
	"context"
	"errors"
	"kanbano-api/internal/brevo"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/models"
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
	workspaceRepo   *repository.WorkspaceRepository
	columnRepo      *repository.ColumnRepository
	taskRepo        *repository.TaskRepository
	accessGrant     *repository.AccessGrantRepository
	store           *storage.Client
	mailer          *brevo.Client
	invitationTplID int
	frontendBaseURL string
}

type OrganisationHandlerConfig struct {
	Repo            *repository.OrganisationRepository
	UserRepo        *repository.UserRepository
	WorkspaceRepo   *repository.WorkspaceRepository
	ColumnRepo      *repository.ColumnRepository
	TaskRepo        *repository.TaskRepository
	AccessGrant     *repository.AccessGrantRepository
	Store           *storage.Client
	Mailer          *brevo.Client
	InvitationTplID int
	FrontendBaseURL string
}

func NewOrganisationHandler(cfg OrganisationHandlerConfig) *OrganisationHandler {
	return &OrganisationHandler{
		repo:            cfg.Repo,
		userRepo:        cfg.UserRepo,
		workspaceRepo:   cfg.WorkspaceRepo,
		columnRepo:      cfg.ColumnRepo,
		taskRepo:        cfg.TaskRepo,
		accessGrant:     cfg.AccessGrant,
		store:           cfg.Store,
		mailer:          cfg.Mailer,
		invitationTplID: cfg.InvitationTplID,
		frontendBaseURL: cfg.FrontendBaseURL,
	}
}

type inviteBody struct {
	Email       string     `json:"email" validate:"required,email"`
	Role        *string    `json:"role,omitempty" validate:"omitempty,oneof=view edit"`
	WorkspaceID *uuid.UUID `json:"workspace_id,omitempty"`
	ColumnID    *uuid.UUID `json:"column_id,omitempty"`
	TaskID      *uuid.UUID `json:"task_id,omitempty"`
}

type updateInvitationBody struct {
	Status string `json:"status" validate:"required,oneof=accepted declined"`
}

type organisationResponse struct {
	ID      uuid.UUID        `json:"id"`
	UserID  uuid.UUID        `json:"user_id"`
	Members []memberResponse `json:"members"`
}

type memberResponse struct {
	ID       uuid.UUID         `json:"id"`
	Name     string            `json:"name"`
	Avatar   *models.AvatarSet `json:"avatar"`
	JoinedAt *time.Time        `json:"joined_at"`
	Role     string            `json:"role"`
}

// Get godoc
// @Summary Get the current user's organisation
// @Tags organisation
// @Produce json
// @Success 200 {object} organisationResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /organisation [get]
func (h *OrganisationHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	organisation, err := h.repo.GetOrganisationWithMembers(r.Context(), userID)
	if handleRepoError(w, r, err, "organisation not found") {
		return
	}

	utils.RespondJSON(w, http.StatusOK, h.response(organisation))
}

func (h *OrganisationHandler) response(org models.Organisation) organisationResponse {
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
	return organisationResponse{ID: org.ID, UserID: org.UserID, Members: members}
}

// Invite godoc
// @Summary Invite a member
// @Description Sends an organization-level invitation, or a scoped invitation when workspace_id, column_id, and task_id are all provided together.
// @Tags organization
// @Accept json
// @Produce json
// @Param body body inviteBody true "Invitation to create"
// @Success 201 {object} utils.CreateResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /organisation/invitations [post]
func (h *OrganisationHandler) Invite(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	body, ok := utils.DecodeAndValidate[inviteBody]("OrganisationHandler.Invite", w, r)
	if !ok {
		return
	}

	isOrgInvite := body.WorkspaceID == nil && body.ColumnID == nil && body.TaskID == nil
	isTaskInvite := body.WorkspaceID != nil && body.ColumnID != nil && body.TaskID != nil
	if !isOrgInvite && !isTaskInvite {
		badRequest(w, r, "workspace_id, column_id and task_id must all be provided together")
		return
	}

	if isTaskInvite && !h.validateTaskInviteTarget(w, r, userID, body) {
		return
	}

	org, err := h.repo.GetOrganisationWithMembers(r.Context(), userID)
	if handleRepoError(w, r, err, "organisation not found") {
		return
	}

	invitation, err := h.repo.CreateInvitation(r.Context(), repository.InvitationParams{
		OrganisationID: org.ID,
		Email:          body.Email,
		InvitedBy:      userID,
		WorkspaceID:    body.WorkspaceID,
		ColumnID:       body.ColumnID,
		TaskID:         body.TaskID,
		Role:           body.Role,
	})
	if err != nil {
		if errors.Is(err, repository.ErrInvitationAlreadyPending) {
			conflict(w, r, "an invitation is already pending for this email")
			return
		}
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

// validateTaskInviteTarget checks that the column and task targeted by an
// invitation exist and are accessible to the inviter. Writes an error
// response and returns false if either check fails.
func (h *OrganisationHandler) validateTaskInviteTarget(w http.ResponseWriter, r *http.Request, userID uuid.UUID, body inviteBody) bool {
	colExists, err := h.columnRepo.Exists(r.Context(), *body.ColumnID, *body.WorkspaceID)
	if err != nil {
		serverError(w, r, err)
		return false
	}
	if !colExists {
		notFound(w, r, "column not found")
		return false
	}

	hasColumnAccess, err := h.accessGrant.HasColumnAccess(r.Context(), *body.WorkspaceID, *body.ColumnID, userID)
	if err != nil {
		serverError(w, r, err)
		return false
	}
	if !hasColumnAccess {
		notFound(w, r, "column not found")
		return false
	}

	taskExists, err := h.taskRepo.Exists(r.Context(), *body.TaskID, *body.ColumnID)
	if err != nil {
		serverError(w, r, err)
		return false
	}
	if !taskExists {
		notFound(w, r, "task not found")
		return false
	}

	return true
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
// @Summary List invitations sent by the organisation
// @Tags organisation
// @Produce json
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.OrganisationInvitation
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /organisation/invitations/sent [get]
func (h *OrganisationHandler) SentInvitations(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	limit, offset, ok := parsePagination(w, r)
	if !ok {
		return
	}

	org, err := h.repo.GetOrganisationWithMembers(r.Context(), userID)
	if handleRepoError(w, r, err, "organisation not found") {
		return
	}

	invitations, err := h.repo.ListSentInvitations(r.Context(), org.ID, limit, offset)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, invitations)
}

// ReceivedInvitations godoc
// @Summary List invitations received by the current user
// @Tags organisation
// @Produce json
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.OrganisationInvitation
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /organisation/invitations/received [get]
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
// @Summary Accept or decline an invitation
// @Tags organisation
// @Accept json
// @Produce json
// @Param id path string true "Invitation ID"
// @Param body body updateInvitationBody true "New status"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /organisation/invitations/{id} [patch]
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

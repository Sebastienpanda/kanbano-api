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
	Role  *string `json:"role,omitempty" validate:"omitempty,oneof=view edit"`
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

type memberProfileResponse struct {
	ID         uuid.UUID              `json:"id"`
	Email      string                 `json:"email"`
	Name       string                 `json:"name"`
	Avatar     *models.AvatarSet      `json:"avatar"`
	Role       string                 `json:"role"`
	Workspaces []models.WorkspaceRole `json:"workspaces"`
}

// MemberProfile godoc
// @Summary Get a member's profile within the organisation
// @Description Returns the member's identity, organisation role, and effective role on every workspace of the organisation. The caller must belong to the same organisation as the target member.
// @Tags organisation
// @Produce json
// @Param id path string true "Member ID (user ID)"
// @Success 200 {object} memberProfileResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /organisation/members/{id}/profile [get]
func (h *OrganisationHandler) MemberProfile(w http.ResponseWriter, r *http.Request) {
	callerID := userIDFromContext(r)

	memberID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	profile, err := h.repo.GetMemberProfile(r.Context(), callerID, memberID)
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
// @Summary Invite a member
// @Description Sends an organisation-level invitation by email.
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

	org, err := h.repo.GetOrganisationWithMembers(r.Context(), userID)
	if handleRepoError(w, r, err, "organisation not found") {
		return
	}

	invitation, err := h.repo.CreateInvitation(r.Context(), repository.InvitationParams{
		OrganisationID: org.ID,
		Email:          body.Email,
		InvitedBy:      userID,
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

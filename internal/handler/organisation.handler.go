package handler

import (
	"net/http"
	"time"

	"kanbano-api/internal/models"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/storage"
	"kanbano-api/internal/utils"

	"github.com/google/uuid"
)

type OrganisationHandler struct {
	repo  *repository.OrganisationRepository
	store *storage.Client
}

func NewOrganisationHandler(repo *repository.OrganisationRepository, store *storage.Client) *OrganisationHandler {
	return &OrganisationHandler{repo: repo, store: store}
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
}

func (h *OrganisationHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	organisation, err := h.repo.GetOrganisationWithMembers(r.Context(), userID)
	if handleRepoError(w, err, "organisation not found") {
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
		}
	}
	return organisationResponse{ID: org.ID, UserID: org.UserID, Members: members}
}

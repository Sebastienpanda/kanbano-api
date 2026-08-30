package handler

import (
	"kanbano-api/internal/repository"
	"net/http"
)

type OrganisationHandler struct {
	repo *repository.OrganisationRepository
}

func NewOrganisationHandler(repo *repository.OrganisationRepository) *OrganisationHandler {
	return &OrganisationHandler{repo: repo}
}

func (h *OrganisationHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	organisation, err := h.repo.GetByUserWithWorkspaces(r.Context(), userID)
	if handleRepoError(w, err, "organisation not found") {
		return
	}

	writeJSON(w, http.StatusOK, organisation)
}

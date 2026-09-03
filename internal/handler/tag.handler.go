package handler

import (
	"kanbano-api/internal/repository"
	"kanbano-api/internal/utils"
	"net/http"
)

type TagHandler struct {
	repo *repository.TagRepository
}

type createTagBody struct {
	Name  string  `json:"name" validate:"required,min=1,max=50"`
	Color *string `json:"color" validate:"omitempty,max=50"`
}

type updateTagBody struct {
	Name  *string `json:"name" validate:"omitempty,min=1,max=50"`
	Color *string `json:"color" validate:"omitempty,max=50"`
}

func NewTagHandler(repo *repository.TagRepository) *TagHandler {
	return &TagHandler{repo: repo}
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	tags, err := h.repo.List(r.Context(), userID)
	if err != nil {
		serverError(w, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, tags)
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	body, ok := utils.DecodeAndValidate[createTagBody]("TagHandler.Create", w, r)
	if !ok {
		return
	}

	tag, err := h.repo.Create(r.Context(), body.Name, body.Color, userID)
	if err != nil {
		serverError(w, err)
		return
	}
	utils.RespondCreated(w, &tag.ID)
}

func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	tagID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := utils.DecodeAndValidate[updateTagBody]("TagHandler.Update", w, r)
	if !ok {
		return
	}

	_, err := h.repo.Update(r.Context(), tagID, userID, body.Name, body.Color)
	if handleRepoError(w, err, "tag not found") {
		return
	}

	utils.RespondUpdated(w)
}

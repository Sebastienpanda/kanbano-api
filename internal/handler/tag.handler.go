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
	Color *string `json:"color,omitempty" validate:"omitempty,max=50"`
}

type updateTagBody struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=1,max=50"`
	Color *string `json:"color,omitempty" validate:"omitempty,max=50"`
}

func NewTagHandler(repo *repository.TagRepository) *TagHandler {
	return &TagHandler{repo: repo}
}

// List godoc
// @Summary List tags
// @Tags tag
// @Produce json
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.Tag
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /tags [get]
func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	limit, offset, ok := parsePagination(w, r)
	if !ok {
		return
	}

	tags, err := h.repo.List(r.Context(), userID, limit, offset)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, tags)
}

// Create godoc
// @Summary Create a tag
// @Tags tag
// @Accept json
// @Produce json
// @Param body body createTagBody true "Tag to create"
// @Success 201 {object} utils.CreateResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /tags [post]
func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	body, ok := utils.DecodeAndValidate[createTagBody]("TagHandler.Create", w, r)
	if !ok {
		return
	}

	tag, err := h.repo.Create(r.Context(), body.Name, body.Color, userID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	utils.RespondCreated(w, &tag.ID)
}

// Update godoc
// @Summary Update a tag
// @Tags tag
// @Accept json
// @Produce json
// @Param id path string true "Tag ID"
// @Param body body updateTagBody true "Fields to update"
// @Success 200 {object} utils.UpdateResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 422 {object} map[string]any
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /tags/{id} [patch]
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
	if handleRepoError(w, r, err, "tag not found") {
		return
	}

	utils.RespondUpdated(w)
}

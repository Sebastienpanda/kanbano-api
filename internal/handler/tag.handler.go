package handler

import (
	"kanbano-api/internal/repository"
	"net/http"
)

type TagHandler struct {
	repo *repository.TagRepository
}

type createTagBody struct {
	Name  string  `json:"name"`
	Color *string `json:"color"`
}

type updateTagBody struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
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

	writeJSON(w, http.StatusOK, tags)
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	body, ok := decodeJSON[createTagBody](w, r)
	if !ok {
		return
	}
	if body.Name == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tag, err := h.repo.Create(r.Context(), body.Name, body.Color, userID)
	respondCreated(w, tag, err)
}

func (h *TagHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	tagID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	body, ok := decodeJSON[updateTagBody](w, r)
	if !ok {
		return
	}

	tag, err := h.repo.Update(r.Context(), tagID, userID, body.Name, body.Color)
	if handleRepoError(w, err, "tag not found") {
		return
	}

	writeJSON(w, http.StatusOK, tag)
}

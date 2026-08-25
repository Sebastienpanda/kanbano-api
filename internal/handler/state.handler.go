package handler

import (
	"kanbano-api/internal/repository"
	"net/http"
)

type StateHandler struct {
	repo *repository.StateRepository
}

type createStateBody struct {
	Name  string  `json:"name"`
	Color *string `json:"color"`
}

func NewStateHandler(repo *repository.StateRepository) *StateHandler {
	return &StateHandler{repo: repo}
}

func (h *StateHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	states, err := h.repo.List(r.Context(), userID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, states)
}

func (h *StateHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r)

	body, ok := decodeJSON[createStateBody](w, r)
	if !ok {
		return
	}
	if body.Name == "" {
		http.Error(w, "body invalide", http.StatusBadRequest)
		return
	}

	state, err := h.repo.Create(r.Context(), body.Name, body.Color, userID)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, state)
}

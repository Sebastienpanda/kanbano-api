package handler

import (
	"encoding/json"
	"kanbano-api/internal/repository"
	"net/http"
)

type StateHandler struct {
	repo *repository.StateRepository
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

	var body struct {
		Name  string  `json:"name"`
		Color *string `json:"color"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)

	if err != nil || body.Name == "" {
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

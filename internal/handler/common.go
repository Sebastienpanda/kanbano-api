package handler

import (
	"encoding/json"
	"errors"
	"kanbano-api/internal/middleware"
	"kanbano-api/internal/repository"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func userIDFromContext(r *http.Request) uuid.UUID {
	return uuid.MustParse(r.Context().Value(middleware.UserIDKey).(string))
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		http.Error(w, "invalid "+param, http.StatusBadRequest)
		return uuid.UUID{}, false
	}
	return id, true
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var body T
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		var zero T
		return zero, false
	}
	return body, true
}

func requireWorkspace(w http.ResponseWriter, r *http.Request, workspaceRepo *repository.WorkspaceRepository) (userID, workspaceID uuid.UUID, ok bool) {
	userID = userIDFromContext(r)

	workspaceID, ok = parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	exists, err := workspaceRepo.Exists(r.Context(), workspaceID, userID)
	if err != nil {
		serverError(w, err)
		return userID, workspaceID, false
	}
	if !exists {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return userID, workspaceID, false
	}

	return userID, workspaceID, true
}

func handleRepoError(w http.ResponseWriter, err error, notFoundMsg string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, notFoundMsg, http.StatusNotFound)
		return true
	}
	serverError(w, err)
	return true
}

func respondCreated[T any](w http.ResponseWriter, result T, err error) {
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("internal server error: %v", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

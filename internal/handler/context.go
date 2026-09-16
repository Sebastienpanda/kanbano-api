package handler

import (
	"kanbano-api/internal/middleware"
	"kanbano-api/internal/repository"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func userIDFromContext(r *http.Request) uuid.UUID {
	// middleware.AuthRequired always sets UserIDKey to a valid UUID string
	// before a handler runs; a panic here signals a middleware wiring bug.
	//nolint:forcetypeassert,errcheck // invariant guaranteed by AuthRequired
	return uuid.MustParse(r.Context().Value(middleware.UserIDKey).(string))
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		badRequest(w, r, "invalid "+param)
		return uuid.UUID{}, false
	}
	return id, true
}

func requireWorkspace(w http.ResponseWriter, r *http.Request, workspaceRepo *repository.WorkspaceRepository) (userID, workspaceID uuid.UUID, ok bool) {
	userID = userIDFromContext(r)

	workspaceID, ok = parseUUIDParam(w, r, "id")
	if !ok {
		return userID, workspaceID, ok
	}

	exists, err := workspaceRepo.Exists(r.Context(), workspaceID, userID)
	if err != nil {
		serverError(w, r, err)
		return userID, workspaceID, false
	}
	if !exists {
		notFound(w, r, "workspace not found")
		return userID, workspaceID, false
	}

	return userID, workspaceID, true
}

package handler

import (
	"kanbano-api/internal/middleware"
	"kanbano-api/internal/permissions"
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

// requireWorkspace parses the {id} workspace param and resolves the
// caller's roles on it, along with their guest role on the {taskId} task
// when the route has one. Writes a 404 and returns false if the workspace
// does not exist for the caller (permissions.CanSeeWorkspace).
func requireWorkspace(w http.ResponseWriter, r *http.Request, roleRepo *repository.RoleRepository) (userID, workspaceID uuid.UUID, access permissions.Access, ok bool) {
	userID = userIDFromContext(r)

	workspaceID, ok = parseUUIDParam(w, r, "id")
	if !ok {
		return userID, workspaceID, access, false
	}

	taskID := uuid.Nil
	if chi.URLParam(r, "taskId") != "" {
		taskID, ok = parseUUIDParam(w, r, "taskId")
		if !ok {
			return userID, workspaceID, access, false
		}
	}

	access, err := roleRepo.ResolveAccess(r.Context(), workspaceID, taskID, userID)
	if err != nil {
		serverError(w, r, err)
		return userID, workspaceID, access, false
	}
	if !permissions.CanSeeWorkspace(access) {
		notFound(w, r, "workspace not found")
		return userID, workspaceID, access, false
	}

	return userID, workspaceID, access, true
}

// requirePermission writes a 403 with message and returns false when the
// caller is not allowed.
func requirePermission(w http.ResponseWriter, r *http.Request, allowed bool, message string) bool {
	if !allowed {
		forbidden(w, r, message)
	}
	return allowed
}

// Messages of the 403 answered by requirePermission.
const (
	errEditAccessRequired   = "edit access required"
	errOwnerOrAdminRequired = "organisation owner or admin required"
	errOwnerRequired        = "organisation owner required"
	errGuestLimitedToTasks  = "guest access is limited to shared tasks"
)

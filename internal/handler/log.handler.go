package handler

import (
	"kanbano-api/internal/repository"
	"kanbano-api/internal/utils"
	"net/http"
)

type LogHandler struct {
	repo *repository.LogRepository
}

func NewLogHandler(repo *repository.LogRepository) *LogHandler {
	return &LogHandler{repo: repo}
}

// List godoc
// @Summary List request/error logs (admin only)
// @Tags log
// @Produce json
// @Param level query string false "Log level" Enums(info, warning, error)
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.Log
// @Failure 400 {object} ErrorResponse "invalid level | invalid limit | invalid offset"
// @Failure 401 {object} UnauthorizedResponse "Missing or invalid token (text/plain)"
// @Failure 403 {object} AdminForbiddenResponse "Caller is not an administrator (text/plain)"
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Failure 500 {object} InternalErrorResponse
// @Security BearerAuth
// @Router /admin/logs [get]
func (h *LogHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var level *string
	if raw := query.Get("level"); raw != "" {
		if raw != "info" && raw != "warning" && raw != "error" {
			badRequest(w, r, "invalid level")
			return
		}
		level = &raw
	}

	limit, offset, ok := parsePagination(w, r)
	if !ok {
		return
	}

	logs, err := h.repo.List(r.Context(), level, limit, offset)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, logs)
}

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
// @Param level query string false "info, warning or error"
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Page offset (default 0)"
// @Success 200 {array} models.Log
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /admin/logs [get]
func (h *LogHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var level *string
	if raw := query.Get("level"); raw != "" {
		if raw != "info" && raw != "warning" && raw != "error" {
			badRequest(w, "invalid level")
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

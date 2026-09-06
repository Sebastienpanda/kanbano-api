package handler

import (
	"kanbano-api/internal/repository"
	"kanbano-api/internal/utils"
	"net/http"
	"strconv"
)

const (
	defaultLogLimit = 50
	maxLogLimit     = 200
)

type LogHandler struct {
	repo *repository.LogRepository
}

func NewLogHandler(repo *repository.LogRepository) *LogHandler {
	return &LogHandler{repo: repo}
}

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

	limit := defaultLogLimit
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			badRequest(w, "invalid limit")
			return
		}
		limit = parsed
	}
	if limit > maxLogLimit {
		limit = maxLogLimit
	}

	offset := 0
	if raw := query.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			badRequest(w, "invalid offset")
			return
		}
		offset = parsed
	}

	logs, err := h.repo.List(r.Context(), level, limit, offset)
	if err != nil {
		serverError(w, r, err)
		return
	}

	utils.RespondJSON(w, http.StatusOK, logs)
}

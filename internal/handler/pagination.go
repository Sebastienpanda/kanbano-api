package handler

import (
	"net/http"
	"strconv"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 200
)

func parsePagination(w http.ResponseWriter, r *http.Request) (limit, offset int, ok bool) {
	query := r.URL.Query()

	limit = defaultPageLimit
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			badRequest(w, "invalid limit")
			return 0, 0, false
		}
		limit = parsed
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	offset = 0
	if raw := query.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			badRequest(w, "invalid offset")
			return 0, 0, false
		}
		offset = parsed
	}

	return limit, offset, true
}

package handler

import (
	"errors"
	"kanbano-api/internal/utils"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func badRequest(w http.ResponseWriter, msg string) {
	utils.RespondError(w, http.StatusBadRequest, msg)
}

func notFound(w http.ResponseWriter, msg string) {
	utils.RespondError(w, http.StatusNotFound, msg)
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("internal server error: %v", err)
	utils.RespondError(w, http.StatusInternalServerError, "internal server error")
}

// handleRepoError turns a repository error into a 404 (pgx.ErrNoRows) or 500.
// Returns true when an error was handled and the caller should stop.
func handleRepoError(w http.ResponseWriter, err error, notFoundMsg string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		notFound(w, notFoundMsg)
		return true
	}
	serverError(w, err)
	return true
}

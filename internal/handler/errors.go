package handler

import (
	"errors"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/middleware"
	"kanbano-api/internal/utils"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/samber/oops"
)

const errDomain = "handler"

func badRequest(w http.ResponseWriter, r *http.Request, msg string) {
	middleware.SetErrorDetail(r, msg)
	utils.RespondError(w, http.StatusBadRequest, msg)
}

func notFound(w http.ResponseWriter, r *http.Request, msg string) {
	middleware.SetErrorDetail(r, msg)
	utils.RespondError(w, http.StatusNotFound, msg)
}

func forbidden(w http.ResponseWriter, r *http.Request, msg string) {
	middleware.SetErrorDetail(r, msg)
	utils.RespondError(w, http.StatusForbidden, msg)
}

func conflict(w http.ResponseWriter, r *http.Request, msg string) {
	middleware.SetErrorDetail(r, msg)
	utils.RespondError(w, http.StatusConflict, msg)
}

func unprocessableEntity(w http.ResponseWriter, r *http.Request, msg string) {
	middleware.SetErrorDetail(r, msg)
	utils.RespondError(w, http.StatusUnprocessableEntity, msg)
}

func serverError(w http.ResponseWriter, r *http.Request, err error) {
	wrapped := oops.
		In(errDomain).
		Code("internal_server_error").
		With("path", r.URL.Path).
		With("method", r.Method).
		Public("internal server error").
		Wrapf(err, "unhandled error in request")

	logging.Logger.Error("internal server error", slog.Any("error", wrapped))
	middleware.SetErrorDetailFromError(r, wrapped)
	utils.RespondError(w, http.StatusInternalServerError, oops.GetPublic(wrapped, "internal server error"))
}

func safeGo(fn func()) {
	go func() {
		err := oops.
			In(errDomain).
			Code("panic_recovered").
			Recover(fn)
		if err != nil {
			logging.Logger.Error("recovered panic in background task", slog.Any("error", err))
		}
	}()
}

func handleRepoError(w http.ResponseWriter, r *http.Request, err error, notFoundMsg string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		notFound(w, r, notFoundMsg)
		return true
	}
	serverError(w, r, err)
	return true
}

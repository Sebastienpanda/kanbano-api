package handler

import (
	"context"
	"errors"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/middleware"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/utils"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var logRepo *repository.LogRepository

func InitLogRepository(repo *repository.LogRepository) {
	logRepo = repo
}

func badRequest(w http.ResponseWriter, msg string) {
	utils.RespondError(w, http.StatusBadRequest, msg)
}

func notFound(w http.ResponseWriter, msg string) {
	utils.RespondError(w, http.StatusNotFound, msg)
}

func conflict(w http.ResponseWriter, msg string) {
	utils.RespondError(w, http.StatusConflict, msg)
}

func unprocessableEntity(w http.ResponseWriter, msg string) {
	utils.RespondError(w, http.StatusUnprocessableEntity, msg)
}

func serverError(w http.ResponseWriter, r *http.Request, err error) {
	logging.Logger.Error("internal server error", slog.Any("error", err))
	logRequestError(r, err)
	utils.RespondError(w, http.StatusInternalServerError, "internal server error")
}

func logRequestError(r *http.Request, err error) {
	if logRepo == nil {
		return
	}

	source := requestSource(r)
	userID := userIDFromRequestContext(r)
	requestID := requestIDFromRequest(r)

	safeGo(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if insertErr := logRepo.Insert(ctx, "error", err.Error(), source, userID, requestID, nil); insertErr != nil {
			logging.Logger.Error("failed to persist error log", slog.Any("error", insertErr))
		}
	})
}

func safeGo(fn func()) {
	go func() {
		defer func() {
			if p := recover(); p != nil {
				logging.Logger.Error("recovered panic in background task", slog.Any("panic", p))
			}
		}()
		fn()
	}()
}

func requestSource(r *http.Request) string {
	path := r.URL.Path
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			path = pattern
		}
	}
	return r.Method + " " + path
}

func userIDFromRequestContext(r *http.Request) *uuid.UUID {
	raw, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

func requestIDFromRequest(r *http.Request) *uuid.UUID {
	return middleware.RequestIDFromContext(r.Context())
}

func handleRepoError(w http.ResponseWriter, r *http.Request, err error, notFoundMsg string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		notFound(w, notFoundMsg)
		return true
	}
	serverError(w, r, err)
	return true
}

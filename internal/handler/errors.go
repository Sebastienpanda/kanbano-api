package handler

import (
	"context"
	"errors"
	"kanbano-api/internal/middleware"
	"kanbano-api/internal/repository"
	"kanbano-api/internal/utils"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var logRepo *repository.LogRepository

// InitLogRepository wires the repository used to persist error/warning logs.
// Must be called once at startup, before the server starts handling requests.
func InitLogRepository(repo *repository.LogRepository) {
	logRepo = repo
}

func badRequest(w http.ResponseWriter, msg string) {
	utils.RespondError(w, http.StatusBadRequest, msg)
}

func notFound(w http.ResponseWriter, msg string) {
	utils.RespondError(w, http.StatusNotFound, msg)
}

func serverError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("internal server error: %v", err)
	logRequestError(r, err)
	utils.RespondError(w, http.StatusInternalServerError, "internal server error")
}

// logRequestError persists an error-level log entry asynchronously. It must
// not use r.Context() since it is cancelled once the response is written.
func logRequestError(r *http.Request, err error) {
	if logRepo == nil {
		return
	}

	source := requestSource(r)
	userID := userIDFromRequestContext(r)
	requestID := requestIDFromRequest(r)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if insertErr := logRepo.Insert(ctx, "error", err.Error(), source, userID, requestID, nil); insertErr != nil {
			log.Printf("warning: failed to persist error log: %v", insertErr)
		}
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

// handleRepoError turns a repository error into a 404 (pgx.ErrNoRows) or 500.
// Returns true when an error was handled and the caller should stop.
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

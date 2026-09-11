package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"kanbano-api/internal/repository"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

var requestLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

var slowRequestThreshold time.Duration = time.Second

// InitSlowRequestThreshold sets the latency threshold (in milliseconds)
// above which a request is persisted as a warning log. Must be called once
// at startup, before the server starts handling requests.
func InitSlowRequestThreshold(ms int) {
	if ms <= 0 {
		ms = 1000
	}
	slowRequestThreshold = time.Duration(ms) * time.Millisecond
}

func NewRequestLogger(logRepo *repository.LogRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			latency := time.Since(start)

			requestID := ""
			if id := RequestIDFromContext(r.Context()); id != nil {
				requestID = id.String()
			}

			attrs := []any{
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int64("latency_ms", latency.Milliseconds()),
			}

			if userID, ok := r.Context().Value(UserIDKey).(string); ok && userID != "" {
				attrs = append(attrs, slog.String("user_id", userID))
			}

			switch {
			case status >= 500:
				requestLogger.Error("http_request", attrs...)
			case status >= 400:
				requestLogger.Warn("http_request", attrs...)
			default:
				requestLogger.Info("http_request", attrs...)
			}

			if status < 500 && latency >= slowRequestThreshold {
				logSlowRequest(logRepo, r, latency, status)
			}
		})
	}
}

func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return ""
	}
	return rctx.RoutePattern()
}

func logSlowRequest(logRepo *repository.LogRepository, r *http.Request, latency time.Duration, status int) {
	if logRepo == nil {
		return
	}

	path := r.URL.Path
	if pattern := routePattern(r); pattern != "" {
		path = pattern
	}
	source := r.Method + " " + path

	var userID *uuid.UUID
	if raw, ok := r.Context().Value(UserIDKey).(string); ok && raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			userID = &id
		}
	}

	requestID := RequestIDFromContext(r.Context())

	metadata, err := json.Marshal(map[string]any{
		"latency_ms": latency.Milliseconds(),
		"status":     status,
	})
	if err != nil {
		metadata = nil
	}

	message := fmt.Sprintf("slow request: %s", latency)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if insertErr := logRepo.Insert(ctx, "warning", message, source, userID, requestID, metadata); insertErr != nil {
			log.Printf("warning: failed to persist slow request log: %v", insertErr)
		}
	}()
}

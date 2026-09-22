package middleware

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/repository"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

var requestLogger = logging.Logger

var slowRequestThreshold = time.Second

// maxErrorBodyCapture bounds the size of the response body captured for
// 5xx errors, to avoid bloating the logs on a large response.
const maxErrorBodyCapture = 2048

// errorBodyWriter captures the bytes written to the response so they can be
// logged in case of a server error (e.g. http.Error called by a handler or
// a dependency like swag/http-swagger).
type errorBodyWriter struct {
	chimiddleware.WrapResponseWriter
	body bytes.Buffer
}

func (w *errorBodyWriter) Write(p []byte) (int, error) {
	if remaining := maxErrorBodyCapture - w.body.Len(); remaining > 0 {
		if remaining > len(p) {
			remaining = len(p)
		}
		w.body.Write(p[:remaining])
	}
	return w.WrapResponseWriter.Write(p)
}

// Hijack exposes http.Hijacker via the underlying ResponseWriter: chi does
// not promote this method through embedding (it isn't in the
// WrapResponseWriter interface), otherwise the websocket upgrade fails with
// "response does not implement http.Hijacker".
func (w *errorBodyWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.WrapResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

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
			ww := &errorBodyWriter{WrapResponseWriter: chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)}
			r, errDetail := WithErrorDetail(r)

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

			errorBody := strings.TrimSpace(ww.body.String())
			if status >= 500 && errorBody != "" {
				attrs = append(attrs, slog.String("error_body", errorBody))
			}

			// A 404 without ErrorDetail comes from no handler (unknown route):
			// internet scanners (/.env, /wp-json, ...) that would flood the logs table.
			isRouteNotFound := status == http.StatusNotFound && errDetail.Message == ""
			if status >= 400 && !isRouteNotFound {
				persistErrorLog(logRepo, r, status, errDetail, errorBody)
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

// persistErrorLog logs to the database every response >= 400, with the
// business message and the file:line of the call site when available (see
// ErrorDetail), falling back to the response body otherwise (e.g. errors
// raised outside the handler helpers, like the auth middleware or the rate
// limiter).
func persistErrorLog(logRepo *repository.LogRepository, r *http.Request, status int, detail *ErrorDetail, errorBody string) {
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

	message := errorBody
	meta := map[string]any{"status": status}
	if detail != nil && detail.Message != "" {
		message = detail.Message
		meta["file"] = detail.File
		meta["line"] = detail.Line
		if detail.Code != "" {
			meta["code"] = detail.Code
		}
		if len(detail.Context) > 0 {
			meta["context"] = detail.Context
		}
	}
	if message == "" {
		message = fmt.Sprintf("HTTP %d", status)
	}
	if errorBody != "" {
		meta["error_body"] = errorBody
	}

	metadata, err := json.Marshal(meta)
	if err != nil {
		metadata = nil
	}

	level := "warning"
	if status >= 500 {
		level = "error"
	}

	//nolint:gosec // intentional detachment from the request context, see logSlowRequest
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if insertErr := logRepo.Insert(ctx, level, message, source, userID, requestID, metadata); insertErr != nil {
			logging.Logger.Error("failed to persist error log", slog.Any("error", insertErr))
		}
	}()
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

	// context.Background() is intentional: this log is fire-and-forget and
	// must outlive the HTTP request (r.Context() would be canceled as soon
	// as the response is sent, before the database insert).
	//nolint:gosec // intentional detachment from the request context
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if insertErr := logRepo.Insert(ctx, "warning", message, source, userID, requestID, metadata); insertErr != nil {
			logging.Logger.Error("failed to persist slow request log", slog.Any("error", insertErr))
		}
	}()
}

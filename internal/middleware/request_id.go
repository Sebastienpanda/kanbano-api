package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const RequestIDKey contextKey = "request_id"

// WithRequestID generates a UUID for the incoming request and stores it in
// the request context, so it can be reused for logging and error tracking.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New()
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext returns the request UUID stored in the context, or
// nil if absent.
func RequestIDFromContext(ctx context.Context) *uuid.UUID {
	id, ok := ctx.Value(RequestIDKey).(uuid.UUID)
	if !ok {
		return nil
	}
	return &id
}

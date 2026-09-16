package middleware

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/samber/oops"
)

const errorDetailKey contextKey = "error_detail"

// ErrorDetail captures the business context of an error (message + file/line
// of the call site, and where applicable the code and structured context of
// an oops error) so that NewRequestLogger can persist a rich log without
// each handler having to handle persistence itself.
type ErrorDetail struct {
	Message string
	File    string
	Line    int
	Code    string
	Context map[string]any
}

// WithErrorDetail attaches an empty, mutable ErrorDetail to the request
// context. Called once by NewRequestLogger, at the top of the chain.
func WithErrorDetail(r *http.Request) (*http.Request, *ErrorDetail) {
	detail := &ErrorDetail{}
	return r.WithContext(context.WithValue(r.Context(), errorDetailKey, detail)), detail
}

// ErrorDetailFromContext returns the ErrorDetail attached by WithErrorDetail,
// or nil if there isn't one (e.g. request handled outside NewRequestLogger).
func ErrorDetailFromContext(r *http.Request) *ErrorDetail {
	detail, _ := r.Context().Value(errorDetailKey).(*ErrorDetail)
	return detail
}

// SetErrorDetail is called by the handlers' error helpers
// (badRequest, notFound, conflict, unprocessableEntity, serverError) to
// record the message and the exact file:line where the error was decided.
func SetErrorDetail(r *http.Request, message string) {
	detail := ErrorDetailFromContext(r)
	if detail == nil {
		return
	}
	detail.Message = message
	if _, file, line, ok := runtime.Caller(2); ok {
		detail.File = filepath.Base(file)
		detail.Line = line
	}
}

// SetErrorDetailFromError does the same job as SetErrorDetail, but for an
// oops error: it additionally enriches the ErrorDetail with the code and
// structured context (attributes .With/.Tags/...) so that this context ends
// up in logs.metadata and stays filterable on the frontend, not just in the
// slog logs. Only called by serverError, at the same nesting level as
// SetErrorDetail (hence the same Caller(2)).
func SetErrorDetailFromError(r *http.Request, err error) {
	detail := ErrorDetailFromContext(r)
	if detail == nil {
		return
	}
	detail.Message = err.Error()
	if _, file, line, ok := runtime.Caller(2); ok {
		detail.File = filepath.Base(file)
		detail.Line = line
	}
	if oopsErr, ok := oops.AsOops(err); ok {
		if code := oopsErr.Code(); code != nil {
			detail.Code = fmt.Sprint(code)
		}
		detail.Context = oopsErr.Context()
	}
}

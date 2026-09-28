package handler

// Reusable error components for the generated OpenAPI spec (swag).
//
// These types are never instantiated: handlers keep writing errors through
// the helpers in errors.go and utils. They only exist so that every
// annotation can reference one shared definition instead of repeating the
// shape and the fixed messages of each error on every route.
//
// Convention in the annotations:
//   - a component whose messages are fixed (401, 403 admin, 415, 429, 500)
//     carries them itself as an enum: the @Failure line needs no description;
//   - route-specific messages (invalid id, xxx not found, ...) are listed in
//     the @Failure description, separated by " | ".

// UnauthorizedResponse is the plain-text (text/plain) body written by the
// auth middleware when the Bearer token is missing or invalid.
type UnauthorizedResponse string // @name UnauthorizedResponse

const (
	UnauthorizedMissingToken UnauthorizedResponse = "missing token"
	UnauthorizedInvalidToken UnauthorizedResponse = "invalid token"
)

// AdminForbiddenResponse is the plain-text (text/plain) body written by the
// admin middleware when the caller is not an administrator.
type AdminForbiddenResponse string // @name AdminForbiddenResponse

const AdminForbidden AdminForbiddenResponse = "forbidden"

// TooManyRequestsResponse is the plain-text (text/plain) body written by the
// per-IP rate limiter (10 req/s, burst 30). It applies to every route.
type TooManyRequestsResponse string // @name TooManyRequestsResponse

const TooManyRequests TooManyRequestsResponse = "too many requests"

// BadRequestResponse is the 400 body of a route that reads a JSON body. It
// holds either "error" (a business message: "could not read body", "could
// not decode body" — malformed JSON or unknown field — or a route-specific
// message such as "invalid id"), or "errors" (validation failures, one
// French message per field, keyed by its JSON name).
type BadRequestResponse struct {
	Error  string            `json:"error,omitempty" example:"could not decode body"`
	Args   []string          `json:"args,omitempty"`
	Errors map[string]string `json:"errors,omitempty" example:"name:name est un champ obligatoire"`
} // @name BadRequestResponse

// UnsupportedMediaTypeResponse is returned by every route that reads a JSON
// body when Content-Type is not exactly "application/json".
type UnsupportedMediaTypeResponse struct {
	Error string   `json:"error" enums:"unexpected content-type"`
	Args  []string `json:"args"`
} // @name UnsupportedMediaTypeResponse

// InternalErrorResponse is returned on any unexpected server error; the
// details are only logged, never exposed.
type InternalErrorResponse struct {
	Error string   `json:"error" enums:"internal server error"`
	Args  []string `json:"args"`
} // @name InternalErrorResponse

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"kanbano-api/internal/logging"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
)

func DecodeJSONBody[T any](fn string, w http.ResponseWriter, r *http.Request) (*T, error) {
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		logging.Logger.Warn("unexpected content-type", slog.String("fn", fn), slog.String("content_type", contentType))
		RespondError(w, http.StatusUnsupportedMediaType, "unexpected content-type")
		return nil, fmt.Errorf("unexpected content-type %q", contentType)
	}

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		logging.Logger.Warn("could not read body", slog.String("fn", fn), slog.Any("error", err))
		RespondError(w, http.StatusBadRequest, "could not read body")
		return nil, fmt.Errorf("could not read body: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(buf))
	dec.DisallowUnknownFields()

	var req T
	err = dec.Decode(&req)
	if err != nil {
		logging.Logger.Warn("could not decode body", slog.String("fn", fn), slog.Any("error", err))
		RespondError(w, http.StatusBadRequest, "could not decode body")
		return nil, fmt.Errorf("could not decode body: %w", err)
	}
	// PostgreSQL rejects a NUL character in a text value: refuse it here
	// rather than failing with a 500 in the query.
	if containsNUL(reflect.ValueOf(req)) {
		logging.Logger.Warn("NUL character in body", slog.String("fn", fn))
		RespondError(w, http.StatusBadRequest, "could not decode body")
		return nil, fmt.Errorf("could not decode body: NUL character")
	}

	return &req, nil
}

// containsNUL reports whether a string reachable from v contains a NUL
// character.
func containsNUL(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return strings.ContainsRune(v.String(), 0)
	case reflect.Pointer, reflect.Interface:
		return !v.IsNil() && containsNUL(v.Elem())
	case reflect.Struct:
		for i := range v.NumField() {
			if containsNUL(v.Field(i)) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if containsNUL(v.Index(i)) {
				return true
			}
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			if containsNUL(k) || containsNUL(v.MapIndex(k)) {
				return true
			}
		}
	}
	return false
}

func DecodeAndValidate[T any](fn string, w http.ResponseWriter, r *http.Request) (T, bool) {
	body, err := DecodeJSONBody[T](fn, w, r)
	if err != nil {
		var zero T
		return zero, false
	}

	if err := validate.Struct(*body); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]any{"errors": validationErrors(err)})
		var zero T
		return zero, false
	}

	return *body, true
}

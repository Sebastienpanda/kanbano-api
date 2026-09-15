package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"kanbano-api/internal/logging"
	"log/slog"
	"net/http"
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

	return &req, nil
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

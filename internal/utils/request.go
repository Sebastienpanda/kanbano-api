package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func DecodeJsonBody[T any](fn string, w http.ResponseWriter, r *http.Request) (*T, error) {
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Printf("%s: unexpected content-type %q", fn, contentType)
		RespondError(w, http.StatusUnsupportedMediaType, "unexpected content-type")
		return nil, fmt.Errorf("unexpected content-type %q", contentType)
	}

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("%s: could not read body: %v", fn, err)
		RespondError(w, http.StatusBadRequest, "could not read body")
		return nil, fmt.Errorf("could not read body: %v", err)
	}

	dec := json.NewDecoder(bytes.NewReader(buf))
	dec.DisallowUnknownFields()

	var req T
	err = dec.Decode(&req)
	if err != nil {
		log.Printf("%s: could not decode body: %v", fn, err)
		RespondError(w, http.StatusBadRequest, "could not decode body")
		return nil, fmt.Errorf("could not decode body: %v", err)
	}

	return &req, nil
}

func DecodeAndValidate[T any](fn string, w http.ResponseWriter, r *http.Request) (T, bool) {
	body, err := DecodeJsonBody[T](fn, w, r)
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

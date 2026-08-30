package utils

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

const CreatedStatus = "created"

type CreateResponse struct {
	ID     *uuid.UUID `json:"id,omitempty"`
	Status string     `json:"status"`
}

func RespondJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	err := enc.Encode(v)
	if err != nil {
		log.Printf("RespondJSON: could not encode response failed: %v", err)
		RespondError(w, http.StatusInternalServerError, "could not write response")
		return
	}
}

func RespondCreated(w http.ResponseWriter, id *uuid.UUID) {
	RespondJSON(w, http.StatusCreated, CreateResponse{Status: CreatedStatus, ID: id})
}

const UpdatedStatus = "updated"

type UpdateResponse struct {
	Status string `json:"status"`
}

func RespondUpdated(w http.ResponseWriter) {
	RespondJSON(w, http.StatusOK, UpdateResponse{Status: UpdatedStatus})
}

func RespondDeleted(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

type ErrorResponse struct {
	Error string   `json:"error"`
	Args  []string `json:"args"`
}

func RespondError(w http.ResponseWriter, statusCode int, errorMessage string, args ...string) {
	RespondJSON(w, statusCode, ErrorResponse{Error: errorMessage, Args: args})
}

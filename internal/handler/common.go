package handler

import (
	"encoding/json"
	"errors"
	"kanbano-api/internal/middleware"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func userIDFromContext(r *http.Request) uuid.UUID {
	return uuid.MustParse(r.Context().Value(middleware.UserIDKey).(string))
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		http.Error(w, param+" invalide", http.StatusBadRequest)
		return uuid.UUID{}, false
	}
	return id, true
}

// decodeJSON décode le body JSON dans un T. En cas d'échec, répond directement
// en 400 et renvoie ok=false.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var body T
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "body invalide", http.StatusBadRequest)
		var zero T
		return zero, false
	}
	return body, true
}

func handleRepoError(w http.ResponseWriter, err error, notFoundMsg string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, notFoundMsg, http.StatusNotFound)
		return true
	}
	serverError(w, err)
	return true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func serverError(w http.ResponseWriter, err error) {
	log.Println("erreur serveur:", err)
	http.Error(w, "erreur serveur", http.StatusInternalServerError)
}

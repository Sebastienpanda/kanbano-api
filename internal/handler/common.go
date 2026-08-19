package handler

import (
	"encoding/json"
	"kanbano-api/internal/middleware"
	"log"
	"net/http"

	"github.com/google/uuid"
)

func userIDFromContext(r *http.Request) uuid.UUID {
	return uuid.MustParse(r.Context().Value(middleware.UserIDKey).(string))
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

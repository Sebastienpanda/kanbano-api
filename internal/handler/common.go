package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"kanbano-api/internal/middleware"
	"kanbano-api/internal/repository"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}

func validationMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "ce champ est requis"
	case "min":
		return fmt.Sprintf("doit faire au moins %s caractères", e.Param())
	case "max":
		return fmt.Sprintf("doit faire au plus %s caractères", e.Param())
	case "len":
		return fmt.Sprintf("doit faire exactement %s caractères", e.Param())
	case "email":
		return "doit être une adresse email valide"
	case "url":
		return "doit être une URL valide"
	case "uuid":
		return "doit être un UUID valide"
	case "oneof":
		return fmt.Sprintf("doit être l'une des valeurs: %s", e.Param())
	case "alphanum":
		return "ne doit contenir que des lettres et des chiffres"
	case "lowercase":
		return "doit être en minuscules"
	default:
		return "valeur invalide"
	}
}

func validationErrors(err error) map[string]string {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return map[string]string{"_": err.Error()}
	}
	out := make(map[string]string, len(verrs))
	for _, e := range verrs {
		out[e.Field()] = validationMessage(e)
	}
	return out
}

func userIDFromContext(r *http.Request) uuid.UUID {
	return uuid.MustParse(r.Context().Value(middleware.UserIDKey).(string))
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		http.Error(w, "invalid "+param, http.StatusBadRequest)
		return uuid.UUID{}, false
	}
	return id, true
}

func decodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var body T
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		var zero T
		return zero, false
	}
	return body, true
}

func decodeAndValidate[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	body, ok := decodeJSON[T](w, r)
	if !ok {
		return body, false
	}
	err := validate.Struct(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"errors": validationErrors(err)})
		var zero T
		return zero, false
	}
	return body, true
}

func requireWorkspace(w http.ResponseWriter, r *http.Request, workspaceRepo *repository.WorkspaceRepository) (userID, workspaceID uuid.UUID, ok bool) {
	userID = userIDFromContext(r)

	workspaceID, ok = parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	exists, err := workspaceRepo.Exists(r.Context(), workspaceID, userID)
	if err != nil {
		serverError(w, err)
		return userID, workspaceID, false
	}
	if !exists {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return userID, workspaceID, false
	}

	return userID, workspaceID, true
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

func respondCreated[T any](w http.ResponseWriter, result T, err error) {
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("internal server error: %v", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

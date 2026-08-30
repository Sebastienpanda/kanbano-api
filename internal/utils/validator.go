package utils

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
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

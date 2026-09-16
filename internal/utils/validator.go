package utils

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/locales/fr"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	fr_translations "github.com/go-playground/validator/v10/translations/fr"
)

var (
	validate = newValidator()
	trans    ut.Translator
)

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	frLocale := fr.New()
	uni := ut.New(frLocale, frLocale)
	trans, _ = uni.GetTranslator("fr")
	if err := fr_translations.RegisterDefaultTranslations(v, trans); err != nil {
		panic("could not register fr validator translations: " + err.Error())
	}

	return v
}

func validationErrors(err error) map[string]string {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return map[string]string{"_": err.Error()}
	}
	out := make(map[string]string, len(verrs))
	for _, e := range verrs {
		out[e.Field()] = e.Translate(trans)
	}
	return out
}

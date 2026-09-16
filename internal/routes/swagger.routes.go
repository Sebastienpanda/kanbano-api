package routes

import (
	"os"

	_ "kanbano-api/docs" // registers the generated swagger spec with httpSwagger

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func SwaggerRoutes(r chi.Router) {
	if os.Getenv("APP_ENV") == "production" {
		return
	}

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/api/v1/swagger/doc.json")))
}

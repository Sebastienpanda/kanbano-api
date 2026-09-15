package main

import (
	"fmt"
	"kanbano-api/internal/routes"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := routes.SetupRouter(nil, nil)

	_ = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		fmt.Printf("%-7s %s\n", method, route)
		return nil
	})
}

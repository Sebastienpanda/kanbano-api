// Package logging fournit le logger slog structuré partagé par toute l'application.
package logging

import (
	"log/slog"
	"os"
)

// Logger est le logger JSON structuré utilisé dans tout le projet.
// Les données variables (erreurs, IDs, valeurs) doivent être passées en
// attributes slog, jamais interpolées dans le message, pour garder des
// messages stables et agrégeables par un outil de log.
var Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

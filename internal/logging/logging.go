// Package logging provides the structured slog logger shared across the whole application.
package logging

import (
	"log/slog"
	"os"
)

// Logger is the structured JSON logger used throughout the project.
// Variable data (errors, IDs, values) must be passed as slog attributes,
// never interpolated into the message, to keep messages stable and
// aggregatable by a log tool.
var Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

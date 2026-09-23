package handler

import (
	"kanbano-api/internal/utils"
	"net/http"
)

// Health is the liveness probe used by the host. It deliberately does not
// ping the database, so that probes do not keep the Neon compute awake.
func Health(w http.ResponseWriter, _ *http.Request) {
	utils.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

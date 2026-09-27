package handler

import (
	"kanbano-api/internal/utils"
	"net/http"
)

// Health is the liveness probe used by the host. It deliberately does not
// ping the database, so that probes do not keep the Neon compute awake.
//
// @Summary Liveness probe
// @Description Always answers 200 while the process is up; does not ping the database.
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 429 {object} TooManyRequestsResponse "Rate limit exceeded (text/plain)"
// @Router /../health [get]
func Health(w http.ResponseWriter, _ *http.Request) {
	utils.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

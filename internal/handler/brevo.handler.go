package handler

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"kanbano-api/internal/logging"
	"kanbano-api/internal/utils"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var discordClient = &http.Client{Timeout: 5 * time.Second}

type BrevoHandler struct {
	webhookSecret string
	discordURL    string
}

func NewBrevoHandler(webhookSecret, discordURL string) *BrevoHandler {
	return &BrevoHandler{webhookSecret: webhookSecret, discordURL: discordURL}
}

type brevoEvent struct {
	Event   string `json:"event"`
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Reason  string `json:"reason"`
	Date    string `json:"date"`
}

// Webhook godoc
// @Summary Brevo transactional email webhook
// @Description Receives Brevo email event callbacks (bounce, spam, etc.), authenticated via a "token" query param, and relays failures to Discord if configured.
// @Tags brevo
// @Accept json
// @Produce json
// @Param token query string true "Shared webhook secret"
// @Param body body brevoEvent true "Brevo event payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 415 {object} utils.ErrorResponse
// @Router /../webhooks/brevo [post]
func (h *BrevoHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	if h.webhookSecret == "" || !secretMatches(r.URL.Query().Get("token"), h.webhookSecret) {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Brevo's payload carries many fields we don't model (id, message-id, ts...),
	// so unlike DecodeJSONBody this decoder must tolerate unknown fields.
	var event brevoEvent
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&event); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "could not decode body")
		return
	}

	logging.Logger.Info("brevo webhook received",
		slog.String("event", sanitizeLogValue(event.Event)),
		slog.String("email", maskEmail(event.Email)),
	)

	if h.discordURL != "" {
		if err := h.notifyDiscord(r.Context(), event); err != nil {
			logging.Logger.Error("failed to notify discord for brevo event", slog.Any("error", err))
		}
	}

	utils.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

var failureEvents = map[string]bool{
	"hard_bounce": true,
	"soft_bounce": true,
	"blocked":     true,
	"error":       true,
	"spam":        true,
	"invalid":     true,
}

const (
	discordColorSuccess = 0x57F287
	discordColorFailure = 0xED4245
	discordColorNeutral = 0x5865F2
)

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordEmbed struct {
	Title  string              `json:"title"`
	Color  int                 `json:"color"`
	Fields []discordEmbedField `json:"fields"`
	Footer struct {
		Text string `json:"text"`
	} `json:"footer"`
	Timestamp string `json:"timestamp,omitempty"`
}

func (h *BrevoHandler) notifyDiscord(ctx context.Context, event brevoEvent) error {
	eventName := sanitizeLogValue(event.Event)

	color := discordColorNeutral
	switch {
	case failureEvents[strings.ToLower(eventName)]:
		color = discordColorFailure
	case eventName != "":
		color = discordColorSuccess
	}

	fields := []discordEmbedField{
		{Name: "Event", Value: valueOrDash(eventName), Inline: true},
		{Name: "Email", Value: valueOrDash(sanitizeLogValue(event.Email)), Inline: true},
	}
	if event.Subject != "" {
		fields = append(fields, discordEmbedField{Name: "Sujet", Value: sanitizeLogValue(event.Subject), Inline: false})
	}
	if event.Reason != "" {
		fields = append(fields, discordEmbedField{Name: "Raison", Value: sanitizeLogValue(event.Reason), Inline: false})
	}
	if event.Date != "" {
		fields = append(fields, discordEmbedField{Name: "Date", Value: sanitizeLogValue(event.Date), Inline: true})
	}

	embed := discordEmbed{
		Title:  "Brevo: " + eventName,
		Color:  color,
		Fields: fields,
	}
	embed.Footer.Text = "Kanbano Log Email"

	body, err := json.Marshal(map[string]any{"embeds": []discordEmbed{embed}})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.discordURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := discordClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func secretMatches(provided, expected string) bool {
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func sanitizeLogValue(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	return strings.ReplaceAll(s, "\r", "")
}

func valueOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func maskEmail(email string) string {
	email = sanitizeLogValue(email)
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}

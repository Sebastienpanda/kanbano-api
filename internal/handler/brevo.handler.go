package handler

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
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

func (h *BrevoHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	if h.webhookSecret == "" || !secretMatches(r.URL.Query().Get("token"), h.webhookSecret) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	event, ok := decodeJSON[brevoEvent](w, r)
	if !ok {
		return
	}

	log.Printf("brevo webhook received: event=%s email=%s", sanitizeLogValue(event.Event), maskEmail(event.Email))

	if h.discordURL != "" {
		if err := h.notifyDiscord(event); err != nil {
			log.Printf("failed to notify discord for brevo event: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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

func (h *BrevoHandler) notifyDiscord(event brevoEvent) error {
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

	resp, err := discordClient.Post(h.discordURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

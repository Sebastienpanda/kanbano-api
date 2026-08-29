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

func (h *BrevoHandler) notifyDiscord(event brevoEvent) error {
	content := fmt.Sprintf("**Brevo: %s**\nEmail: %s", sanitizeLogValue(event.Event), sanitizeLogValue(event.Email))
	if event.Subject != "" {
		content += fmt.Sprintf("\nSujet: %s", sanitizeLogValue(event.Subject))
	}
	if event.Reason != "" {
		content += fmt.Sprintf("\nRaison: %s", sanitizeLogValue(event.Reason))
	}
	if event.Date != "" {
		content += fmt.Sprintf("\nDate: %s", sanitizeLogValue(event.Date))
	}

	body, err := json.Marshal(map[string]string{"content": content})
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

func maskEmail(email string) string {
	email = sanitizeLogValue(email)
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}

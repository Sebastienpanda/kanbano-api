package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

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
	if h.webhookSecret != "" && r.URL.Query().Get("token") != h.webhookSecret {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	event, ok := decodeJSON[brevoEvent](w, r)
	if !ok {
		return
	}

	log.Printf("brevo webhook received: event=%s email=%s", event.Event, event.Email)

	if h.discordURL != "" {
		if err := h.notifyDiscord(event); err != nil {
			log.Printf("failed to notify discord for brevo event: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *BrevoHandler) notifyDiscord(event brevoEvent) error {
	content := fmt.Sprintf("**Brevo: %s**\nEmail: %s", event.Event, event.Email)
	if event.Subject != "" {
		content += fmt.Sprintf("\nSujet: %s", event.Subject)
	}
	if event.Reason != "" {
		content += fmt.Sprintf("\nRaison: %s", event.Reason)
	}
	if event.Date != "" {
		content += fmt.Sprintf("\nDate: %s", event.Date)
	}

	body, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return err
	}

	resp, err := http.Post(h.discordURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}
	return nil
}

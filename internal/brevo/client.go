package brevo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey, http: &http.Client{Timeout: 10 * time.Second}}
}

type recipient struct {
	Email string `json:"email"`
}

type sendTemplateEmailRequest struct {
	To         []recipient    `json:"to"`
	TemplateID int            `json:"templateId"`
	Params     map[string]any `json:"params"`
}

func (c *Client) SendTemplateEmail(ctx context.Context, to string, templateID int, params map[string]any) error {
	if c.apiKey == "" {
		return fmt.Errorf("brevo: missing api key")
	}

	body, err := json.Marshal(sendTemplateEmailRequest{
		To:         []recipient{{Email: to}},
		TemplateID: templateID,
		Params:     params,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("brevo: send email failed with status %d", resp.StatusCode)
	}
	return nil
}

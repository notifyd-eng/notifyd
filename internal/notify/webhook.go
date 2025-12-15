package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/notifyd-eng/notifyd/internal/store"
)

type WebhookSender struct {
	client *http.Client
}

func NewWebhookSender(timeout time.Duration) *WebhookSender {
	return &WebhookSender{
		client: &http.Client{Timeout: timeout},
	}
}

func (w *WebhookSender) Channel() string { return "webhook" }

func (w *WebhookSender) Send(ctx context.Context, n *store.Notification) error {
	payload := map[string]interface{}{
		"id":        n.ID,
		"subject":   n.Subject,
		"body":      n.Body,
		"priority":  n.Priority,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.Recipient, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "notifyd/1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

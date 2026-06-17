package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const httpObserverTimeout = 5 * time.Second

// HTTPObserver отправляет события аудита на удалённый сервер методом POST.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver создаёт наблюдатель, отправляющий события на url.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: httpObserverTimeout},
	}
}

func (o *HTTPObserver) Notify(ctx context.Context, event AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("audit send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("audit remote status: %d", resp.StatusCode)
	}
	return nil
}

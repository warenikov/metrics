package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

const httpObserverTimeout = 5 * time.Second

// HTTPObserver отправляет события аудита на удалённый сервер методом POST.
// При временных сбоях выполняет до 3 повторных попыток с экспоненциальной задержкой.
type HTTPObserver struct {
	url    string
	client *retryablehttp.Client
}

// NewHTTPObserver создаёт наблюдатель, отправляющий события на url.
func NewHTTPObserver(url string) *HTTPObserver {
	c := retryablehttp.NewClient()
	c.RetryMax = 3
	c.HTTPClient = &http.Client{Timeout: httpObserverTimeout}
	c.Logger = nil // отключаем стандартный logger retryablehttp
	return &HTTPObserver{url: url, client: c}
}

// Notify сериализует событие и отправляет его POST-запросом на удалённый URL.
// При сетевых ошибках или статусе 5xx автоматически выполняются повторные попытки.
func (o *HTTPObserver) Notify(ctx context.Context, event AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit marshal: %w", err)
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(data))
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

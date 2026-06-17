package audit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureObserver захватывает полученные события для проверки в тестах.
type captureObserver struct {
	events []AuditEvent
	err    error
}

func (o *captureObserver) Notify(_ context.Context, event AuditEvent) error {
	if o.err != nil {
		return o.err
	}
	o.events = append(o.events, event)
	return nil
}

func TestBroker_Emit_AllObserversCalled(t *testing.T) {
	obs1 := &captureObserver{}
	obs2 := &captureObserver{}
	broker := NewBroker(obs1, obs2)

	event := AuditEvent{TS: 1000, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"}
	broker.Emit(context.Background(), event)

	require.Len(t, obs1.events, 1)
	require.Len(t, obs2.events, 1)
	assert.Equal(t, event, obs1.events[0])
	assert.Equal(t, event, obs2.events[0])
}

func TestBroker_Emit_FailingObserverDoesNotStopOthers(t *testing.T) {
	failing := &captureObserver{err: errors.New("sink error")}
	ok := &captureObserver{}
	broker := NewBroker(failing, ok)

	broker.Emit(context.Background(), AuditEvent{TS: 1, Metrics: []string{"X"}, IPAddress: "1.2.3.4"})

	require.Len(t, ok.events, 1)
}

func TestBroker_Emit_NoObservers(t *testing.T) {
	broker := NewBroker()
	assert.NotPanics(t, func() {
		broker.Emit(context.Background(), AuditEvent{})
	})
}

func TestFileObserver_Notify(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "audit-*.log")
	require.NoError(t, err)
	path := f.Name()
	f.Close()

	obs, err := NewFileObserver(path)
	require.NoError(t, err)
	defer obs.Close()

	events := []AuditEvent{
		{TS: 111, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.168.0.1"},
		{TS: 222, Metrics: []string{"HeapAlloc"}, IPAddress: "10.0.0.1"},
	}

	for _, e := range events {
		require.NoError(t, obs.Notify(context.Background(), e))
	}

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	require.Len(t, lines, 2)

	for i, line := range lines {
		var got AuditEvent
		require.NoError(t, json.Unmarshal([]byte(line), &got))
		assert.Equal(t, events[i], got)
	}
}

func TestFileObserver_NewFileObserver_InvalidPath(t *testing.T) {
	_, err := NewFileObserver("/nonexistent/dir/audit.log")
	assert.Error(t, err)
}

func TestHTTPObserver_Notify_Success(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	obs := NewHTTPObserver(srv.URL)
	event := AuditEvent{TS: 999, Metrics: []string{"PollCount"}, IPAddress: "172.16.0.1"}
	err := obs.Notify(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, "application/json", receivedContentType)

	var got AuditEvent
	require.NoError(t, json.Unmarshal(receivedBody, &got))
	assert.Equal(t, event, got)
}

func TestHTTPObserver_Notify_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	obs := NewHTTPObserver(srv.URL)
	err := obs.Notify(context.Background(), AuditEvent{})
	assert.Error(t, err)
}

func TestHTTPObserver_Notify_InvalidURL(t *testing.T) {
	obs := NewHTTPObserver("http://127.0.0.1:0/no-server")
	err := obs.Notify(context.Background(), AuditEvent{})
	assert.Error(t, err)
}

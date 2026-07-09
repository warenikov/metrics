package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"metrics/internal/audit"
	models "metrics/internal/model"
	"metrics/internal/repository"
	"metrics/internal/service"
)

// captureObserver захватывает события аудита для проверки в тестах.
type captureObserver struct {
	events []audit.AuditEvent
}

func (o *captureObserver) Notify(_ context.Context, event audit.AuditEvent) error {
	o.events = append(o.events, event)
	return nil
}

// mockUpdater реализует MetricsUpdater для изолированного тестирования.
type mockUpdater struct {
	parseAndSaveErr error
	updateBatchErr  error
}

func (m *mockUpdater) ParseAndSave(_ context.Context, _, _, _ string) (models.Metrics, error) {
	return models.Metrics{}, m.parseAndSaveErr
}
func (m *mockUpdater) UpdateBatch(_ context.Context, _ []models.Metrics) error {
	return m.updateBatchErr
}

// mockGetter реализует MetricsGetter.
type mockGetter struct {
	result *models.Metrics
	err    error
	list   []models.Metrics
}

func (m *mockGetter) GetMetrica(_ context.Context, _, _ string) (*models.Metrics, error) {
	return m.result, m.err
}
func (m *mockGetter) GetListMetrics(_ context.Context) ([]models.Metrics, error) {
	return m.list, nil
}

// mockHealth реализует HealthChecker.
type mockHealth struct{ err error }

func (m *mockHealth) PingDB() error { return m.err }

func newTestAdapter(updater MetricsUpdater, getter MetricsGetter, health HealthChecker) *httpAdapter {
	if updater == nil {
		updater = &mockUpdater{}
	}
	if getter == nil {
		getter = &mockGetter{}
	}
	if health == nil {
		health = &mockHealth{}
	}
	return &httpAdapter{updater: updater, getter: getter, health: health}
}

func TestHandler_Update(t *testing.T) {
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	h := newTestAdapter(svc, svc, svc)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{"Успешный gauge", "/update/gauge/Alloc/100.5", http.StatusOK},
		{"Успешный counter", "/update/counter/PollCount/5", http.StatusOK},
		{"Ошибка: неверный тип метрики", "/update/invalid/Alloc/100", http.StatusBadRequest},
		{"Ошибка: нечисловое значение", "/update/gauge/Alloc/none", http.StatusBadRequest},
		{"Ошибка: неверное значение для counter", "/update/counter/Alloc/100.5", http.StatusBadRequest},
		{"Успешно отрицательное число для counter", "/update/counter/Alloc/-100", http.StatusOK},
		{"Выход за int64", "/update/counter/Alloc/9223372036854775808", http.StatusBadRequest},
		{"Выход за отрицательное int64", "/update/counter/Alloc/-9223372036854775908", http.StatusBadRequest},
		{"Выход за float64", "/update/gauge/Alloc/1.8e309", http.StatusBadRequest},
		{"Выход за отрицательное float64", "/update/gauge/Alloc/-1.8e309", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", h.update)

			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_GetMetrica(t *testing.T) {
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	h := newTestAdapter(svc, svc, svc)

	var valCounter int64 = 10
	var valGauge = 10.5
	_, _ = repo.UpdateGauges(context.Background(), models.Metrics{ID: "TestGauge", MType: models.Gauge, Value: &valGauge})
	_, _ = repo.UpdateCounter(context.Background(), models.Metrics{ID: "TestName", MType: models.Counter, Delta: &valCounter})
	_, _ = repo.UpdateCounter(context.Background(), models.Metrics{ID: "TestCounter", MType: models.Counter, Delta: &valCounter})

	tests := []struct {
		name           string
		url            string
		expectedBody   string
		expectedStatus int
	}{
		{name: "Получение существующей метрики", url: "/value/gauge/TestGauge", expectedStatus: http.StatusOK, expectedBody: "10.5"},
		{name: "Повторное получение метрики", url: "/value/gauge/TestGauge", expectedStatus: http.StatusOK, expectedBody: "10.5"},
		{name: "Запрос несуществующей метрики", url: "/value/gauge/Unknown", expectedStatus: http.StatusNotFound, expectedBody: ""},
		{name: "Запрос метрики c неверным типом", url: "/value/gauge/TestName", expectedStatus: http.StatusBadRequest, expectedBody: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/value/{type}/{name}", h.getMetrica)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandler_UpdateJSON(t *testing.T) {
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	h := newTestAdapter(svc, svc, svc)

	r := chi.NewRouter()
	r.Post("/update/", h.updateJSON)

	tests := []struct {
		name           string
		body           string
		expectedType   string
		expectedID     string
		expectedStatus int
	}{
		{
			name:           "Валидный gauge",
			body:           `{"id":"Alloc","type":"gauge","value":100.5}`,
			expectedStatus: http.StatusOK,
			expectedType:   models.Gauge,
			expectedID:     "Alloc",
		},
		{
			name:           "Валидный counter",
			body:           `{"id":"PollCount","type":"counter","delta":5}`,
			expectedStatus: http.StatusOK,
			expectedType:   models.Counter,
			expectedID:     "PollCount",
		},
		{name: "Битый JSON", body: `{bad json}`, expectedStatus: http.StatusBadRequest},
		{name: "Неверный тип метрики", body: `{"id":"x","type":"unknown","value":1.0}`, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				var got models.Metrics
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
				assert.Equal(t, tt.expectedID, got.ID)
				assert.Equal(t, tt.expectedType, got.MType)
			}
		})
	}
}

func TestHandler_GetMetricaJSON(t *testing.T) {
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	h := newTestAdapter(svc, svc, svc)

	gaugeVal := 100.5
	var counterVal int64 = 5
	_, _ = repo.UpdateGauges(context.Background(), models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &gaugeVal})
	_, _ = repo.UpdateCounter(context.Background(), models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &counterVal})

	r := chi.NewRouter()
	r.Post("/value/", h.getMetricaJSON)

	tests := []struct {
		checkValue     func(t *testing.T, m models.Metrics)
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Gauge найден",
			body:           `{"id":"Alloc","type":"gauge"}`,
			expectedStatus: http.StatusOK,
			checkValue: func(t *testing.T, m models.Metrics) {
				require.NotNil(t, m.Value)
				assert.Equal(t, 100.5, *m.Value)
			},
		},
		{
			name:           "Counter найден",
			body:           `{"id":"PollCount","type":"counter"}`,
			expectedStatus: http.StatusOK,
			checkValue: func(t *testing.T, m models.Metrics) {
				require.NotNil(t, m.Delta)
				assert.Equal(t, int64(5), *m.Delta)
			},
		},
		{name: "Не существует", body: `{"id":"Unknown","type":"gauge"}`, expectedStatus: http.StatusNotFound},
		{name: "Неверный тип (gauge как counter)", body: `{"id":"Alloc","type":"counter"}`, expectedStatus: http.StatusBadRequest},
		{name: "Битый JSON", body: `{bad}`, expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				var got models.Metrics
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
				tt.checkValue(t, got)
			}
		})
	}
}

func TestHandler_GetMetricsList(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(repo *repository.MemStorage)
		expectInBody []string
	}{
		{name: "Пустое хранилище", setup: func(repo *repository.MemStorage) {}},
		{
			name: "Есть метрики",
			setup: func(repo *repository.MemStorage) {
				v := 42.0
				var d int64 = 7
				_, _ = repo.UpdateGauges(context.Background(), models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &v})
				_, _ = repo.UpdateCounter(context.Background(), models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &d})
			},
			expectInBody: []string{"Alloc", "PollCount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemStorage()
			tt.setup(repo)
			svc := service.NewMetricsService(repo, nil)
			h := newTestAdapter(svc, svc, svc)

			r := chi.NewRouter()
			r.Get("/", h.getMetricsList)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
			for _, s := range tt.expectInBody {
				assert.Contains(t, w.Body.String(), s)
			}
		})
	}
}

func TestHandler_UpdateBatch(t *testing.T) {
	float64Ptr := func(v float64) *float64 { return &v }
	int64Ptr := func(v int64) *int64 { return &v }

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name: "валидный батч",
			body: func() string {
				batch := []models.Metrics{
					{ID: "Alloc", MType: models.Gauge, Value: float64Ptr(1.5)},
					{ID: "PollCount", MType: models.Counter, Delta: int64Ptr(3)},
				}
				b, _ := json.Marshal(batch)
				return string(b)
			}(),
			expectedStatus: http.StatusOK,
		},
		{name: "пустой батч", body: "[]", expectedStatus: http.StatusOK},
		{name: "невалидный JSON", body: "not-json", expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemStorage()
			svc := service.NewMetricsService(repo, nil)
			h := newTestAdapter(svc, svc, svc)

			r := chi.NewRouter()
			r.Post("/updates/", h.updateBatch)

			req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_PingDB(t *testing.T) {
	tests := []struct {
		pingErr        error
		name           string
		expectedStatus int
	}{
		{name: "БД доступна", pingErr: nil, expectedStatus: http.StatusOK},
		{name: "БД недоступна", pingErr: errors.New("connection refused"), expectedStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestAdapter(nil, nil, &mockHealth{err: tt.pingErr})

			r := chi.NewRouter()
			r.Get("/ping", h.pingDB)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestEmitAudit_IPFromRemoteAddr(t *testing.T) {
	cap := &captureObserver{}
	h := &httpAdapter{broker: audit.NewBroker(cap)}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	h.emitAudit(req, []string{"Alloc"})
	h.broker.Close()

	require.Len(t, cap.events, 1)
	assert.Equal(t, "1.2.3.4", cap.events[0].IPAddress)
	assert.Equal(t, []string{"Alloc"}, cap.events[0].Metrics)
}

func TestEmitAudit_XRealIPOverridesRemoteAddr(t *testing.T) {
	cap := &captureObserver{}
	h := &httpAdapter{broker: audit.NewBroker(cap)}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	req.Header.Set("X-Real-IP", "203.0.113.42")
	h.emitAudit(req, []string{"Frees"})
	h.broker.Close()

	require.Len(t, cap.events, 1)
	assert.Equal(t, "203.0.113.42", cap.events[0].IPAddress)
}

func TestEmitAudit_NilBrokerNoPanic(t *testing.T) {
	h := &httpAdapter{broker: nil}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	assert.NotPanics(t, func() { h.emitAudit(req, []string{"X"}) })
}

func TestHandler_Update_EmitsAudit(t *testing.T) {
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	cap := &captureObserver{}
	h := &httpAdapter{updater: svc, getter: svc, health: svc, broker: audit.NewBroker(cap)}

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.update)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/1.5", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	h.broker.Close()
	require.Len(t, cap.events, 1)
	assert.Equal(t, []string{"Alloc"}, cap.events[0].Metrics)
	assert.Equal(t, "10.0.0.1", cap.events[0].IPAddress)
}

func TestHandler_UpdateJSON_EmitsAudit(t *testing.T) {
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	cap := &captureObserver{}
	h := &httpAdapter{updater: svc, getter: svc, health: svc, broker: audit.NewBroker(cap)}

	r := chi.NewRouter()
	r.Post("/update/", h.updateJSON)

	body := `{"id":"PollCount","type":"counter","delta":3}`
	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.1:8080"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	h.broker.Close()
	require.Len(t, cap.events, 1)
	assert.Equal(t, []string{"PollCount"}, cap.events[0].Metrics)
}

func TestHandler_UpdateBatch_EmitsAuditWithAllNames(t *testing.T) {
	float64Ptr := func(v float64) *float64 { return &v }
	int64Ptr := func(v int64) *int64 { return &v }

	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	cap := &captureObserver{}
	h := &httpAdapter{updater: svc, getter: svc, health: svc, broker: audit.NewBroker(cap)}

	r := chi.NewRouter()
	r.Post("/updates/", h.updateBatch)

	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: float64Ptr(1.0)},
		{ID: "PollCount", MType: models.Counter, Delta: int64Ptr(1)},
	}
	data, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "172.16.0.2:4321"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	h.broker.Close()
	require.Len(t, cap.events, 1)
	assert.ElementsMatch(t, []string{"Alloc", "PollCount"}, cap.events[0].Metrics)
}

func TestHandler_Update_NoAuditOnError(t *testing.T) {
	cap := &captureObserver{}
	h := &httpAdapter{
		updater: &mockUpdater{parseAndSaveErr: service.ErrInvalidValue},
		getter:  &mockGetter{},
		health:  &mockHealth{},
		broker:  audit.NewBroker(cap),
	}

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.update)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/bad", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	h.broker.Close()
	assert.Empty(t, cap.events)
}

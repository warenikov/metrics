package handler

import (
	"context"
	"encoding/json"
	"errors"
	"metrics/internal/repository"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"metrics/internal/model"
	"metrics/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockService реализует интерфейс Service для тестирования хендлера в изоляции.
type mockService struct {
	pingErr error
}

func (m *mockService) PingDB() error { return m.pingErr }
func (m *mockService) ParseAndSave(_ context.Context, mType, id, value string) (models.Metrics, error) {
	return models.Metrics{}, nil
}
func (m *mockService) GetMetrica(_ context.Context, mType, id string) (*models.Metrics, error) {
	return nil, nil
}
func (m *mockService) GetListMetrics(_ context.Context) ([]models.Metrics, error) {
	return nil, nil
}
func (m *mockService) UpdateBatch(_ context.Context, metrics []models.Metrics) error {
	return nil
}

func TestHandler_Update(t *testing.T) {
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	h := NewHandler(svc)

	// 2. Описываем сценарии
	tests := []struct {
		name           string
		url            string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "Успешный gauge",
			url:            "/update/gauge/Alloc/100.5",
			contentType:    models.ContentTypeText,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Успешный counter",
			url:            "/update/counter/PollCount/5",
			contentType:    models.ContentTypeText,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Ошибка: неверный тип метрики",
			url:            "/update/invalid/Alloc/100",
			contentType:    models.ContentTypeText,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Ошибка: нечисловое значение",
			url:            "/update/gauge/Alloc/none",
			contentType:    models.ContentTypeText,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Ошибка: неверное значение для counter",
			url:            "/update/counter/Alloc/100.5",
			contentType:    models.ContentTypeText,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Успешно отрицательное число для counter",
			url:            "/update/counter/Alloc/-100",
			contentType:    models.ContentTypeText,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Успешно: выход за int64",
			url:            "/update/counter/Alloc/9223372036854775808",
			contentType:    models.ContentTypeText,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Успешно: выход за отрицательное int64",
			url:            "/update/counter/Alloc/-9223372036854775908",
			contentType:    models.ContentTypeText,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Успешно: выход за float64",
			url:            "/update/gauge/Alloc/1.8e309",
			contentType:    models.ContentTypeText,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Успешно: выход за отрицательное float64",
			url:            "/update/gauge/Alloc/-1.8e309",
			contentType:    models.ContentTypeText,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", h.Update)

			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_GetMetrica(t *testing.T) {
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	h := NewHandler(svc)

	var valCounter int64 = 10 // Явно указываем int64
	var valGauge = 10.5       // Явно указываем int64
	repo.UpdateGauges(context.Background(), models.Metrics{ID: "TestGauge", MType: models.Gauge, Value: &valGauge})
	repo.UpdateCounter(context.Background(), models.Metrics{ID: "TestName", MType: models.Counter, Delta: &valCounter})
	repo.UpdateCounter(context.Background(), models.Metrics{ID: "TestCounter", MType: models.Counter, Delta: &valCounter})

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Получение существующей метрики",
			url:            "/value/gauge/TestGauge",
			expectedStatus: http.StatusOK,
			expectedBody:   "10.5",
		},
		{
			name:           "Получение существующей метрики",
			url:            "/value/gauge/TestGauge",
			expectedStatus: http.StatusOK,
			expectedBody:   "10.5",
		},
		{
			name:           "Запрос несуществующей метрики",
			url:            "/value/gauge/Unknown",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Запрос метрики c неверным типом",
			url:            "/value/gauge/TestName",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/value/{type}/{name}", h.GetMetrica)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandler_UpdateJSON(t *testing.T) {
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo, nil)
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Post("/update/", h.UpdateJSON)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedType   string
		expectedID     string
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
		{
			name:           "Битый JSON",
			body:           `{bad json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Неверный тип метрики",
			body:           `{"id":"x","type":"unknown","value":1.0}`,
			expectedStatus: http.StatusBadRequest,
		},
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
	h := NewHandler(svc)

	// Предзаполняем хранилище
	gaugeVal := 100.5
	var counterVal int64 = 5
	repo.UpdateGauges(context.Background(), models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &gaugeVal})
	repo.UpdateCounter(context.Background(), models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &counterVal})

	r := chi.NewRouter()
	r.Post("/value/", h.GetMetricaJSON)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		checkValue     func(t *testing.T, m models.Metrics)
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
		{
			name:           "Не существует",
			body:           `{"id":"Unknown","type":"gauge"}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Неверный тип (gauge запрошен как counter)",
			body:           `{"id":"Alloc","type":"counter"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Битый JSON",
			body:           `{bad}`,
			expectedStatus: http.StatusBadRequest,
		},
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
		{
			name:  "Пустое хранилище",
			setup: func(repo *repository.MemStorage) {},
		},
		{
			name: "Есть метрики",
			setup: func(repo *repository.MemStorage) {
				v := 42.0
				var d int64 = 7
				repo.UpdateGauges(context.Background(), models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &v})
				repo.UpdateCounter(context.Background(), models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &d})
			},
			expectInBody: []string{"Alloc", "PollCount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemStorage()
			tt.setup(repo)
			svc := service.NewMetricsService(repo, nil)
			h := NewHandler(svc)

			r := chi.NewRouter()
			r.Get("/", h.GetMetricsList)

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
		{
			name:           "пустой батч",
			body:           "[]",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "невалидный JSON",
			body:           "not-json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemStorage()
			svc := service.NewMetricsService(repo, nil)
			h := NewHandler(svc)

			r := chi.NewRouter()
			r.Post("/updates/", h.UpdateBatch)

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
		name           string
		pingErr        error
		expectedStatus int
	}{
		{
			name:           "БД доступна",
			pingErr:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "БД недоступна",
			pingErr:        errors.New("connection refused"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockService{pingErr: tt.pingErr}
			h := NewHandler(svc)

			r := chi.NewRouter()
			r.Get("/ping", h.PingDB)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics/internal/model"
	"metrics/internal/repository"
	"metrics/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestHandler_Update(t *testing.T) {
	repo := storage.NewMemStorage()
	svc := service.NewMetricsService(repo)
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
			name:           "Ошибка: пустой тип контента",
			url:            "/update/gauge/Alloc/100",
			contentType:    "",
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
	repo := storage.NewMemStorage()
	svc := service.NewMetricsService(repo)
	h := NewHandler(svc)

	var valCounter int64 = 10   // Явно указываем int64
	var valGauge float64 = 10.5 // Явно указываем int64
	repo.UpdateGauges(models.Metrics{ID: "TestGauge", MType: models.Gauge, Value: &valGauge})
	repo.UpdateCounter(models.Metrics{ID: "TestName", MType: models.Counter, Delta: &valCounter})
	repo.UpdateCounter(models.Metrics{ID: "TestCounter", MType: models.Counter, Delta: &valCounter})

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
			expectedStatus: http.StatusNotFound,
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

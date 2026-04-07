package repository

import (
	"context"
	"errors"
	"metrics/internal/model"
	"metrics/internal/service"
	"testing"
)

func TestMemStorage_UpdateGauges_Table(t *testing.T) {
	// Вспомогательная функция для создания указателей на float64
	floatPtr := func(f float64) *float64 { return &f }

	tests := []struct {
		name          string
		initialState  map[string]models.Metrics // Предзаполненные данные
		input         models.Metrics            // Что отправляем в UpdateGauges
		expectedValue float64                   // Что ожидаем увидеть в итоге
	}{
		{
			name:          "save new gauge",
			initialState:  nil,
			input:         models.Metrics{ID: "test_new", MType: models.Gauge, Value: floatPtr(10.5)},
			expectedValue: 10.5,
		},
		{
			name: "update existing gauge",
			initialState: map[string]models.Metrics{
				"test_update": {ID: "test_update", MType: models.Gauge, Value: floatPtr(10.5)},
			},
			input:         models.Metrics{ID: "test_update", MType: models.Gauge, Value: floatPtr(20.1)},
			expectedValue: 20.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemStorage()

			if tt.initialState != nil {
				s.metrics = tt.initialState
			}

			_, err := s.UpdateGauges(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			saved, err := s.GetMetrica(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("could not find metric after update: %v", err)
			}

			if *saved.Value != tt.expectedValue {
				t.Errorf("got %f, want %f", *saved.Value, tt.expectedValue)
			}
		})
	}
}

func TestMemStorage_UpdateCounter_Table(t *testing.T) {
	// Хелпер для создания указателя на int64
	intPtr := func(i int64) *int64 { return &i }

	tests := []struct {
		name          string
		initialState  map[string]models.Metrics
		input         models.Metrics
		expectedDelta int64
		wantErr       bool
	}{
		{
			name:          "new counter",
			initialState:  nil,
			input:         models.Metrics{ID: "cnt1", MType: models.Counter, Delta: intPtr(5)},
			expectedDelta: 5,
			wantErr:       false,
		},
		{
			name: "increment existing counter",
			initialState: map[string]models.Metrics{
				"cnt1": {ID: "cnt1", MType: models.Counter, Delta: intPtr(10)},
			},
			input:         models.Metrics{ID: "cnt1", MType: models.Counter, Delta: intPtr(5)},
			expectedDelta: 15, // 10 + 5
			wantErr:       false,
		},
		{
			name:          "error on nil delta",
			initialState:  nil,
			input:         models.Metrics{ID: "cnt1", MType: models.Counter, Delta: nil},
			expectedDelta: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemStorage()
			if tt.initialState != nil {
				s.metrics = tt.initialState
			}

			_, err := s.UpdateCounter(context.Background(), tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("UpdateCounter() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				saved, _ := s.GetMetrica(context.Background(), tt.input)
				if *saved.Delta != tt.expectedDelta {
					t.Errorf("got delta %d, want %d", *saved.Delta, tt.expectedDelta)
				}
			}
		})
	}
}

func TestMemStorage_GetMetrica_Table(t *testing.T) {
	// Хелперы для создания указателей
	floatPtr := func(f float64) *float64 { return &f }

	tests := []struct {
		name         string
		initialState map[string]models.Metrics // Предзаполненная мапа
		input        models.Metrics            // Что ищем
		wantErr      error                     // Какую ошибку ждем
		expectedID   string                    // Какой ID должен вернуться
	}{
		{
			name: "metric exists",
			initialState: map[string]models.Metrics{
				"gauge_1": {ID: "gauge_1", MType: models.Gauge, Value: floatPtr(1.1)},
			},
			input:      models.Metrics{ID: "gauge_1"},
			wantErr:    nil,
			expectedID: "gauge_1",
		},
		{
			name:         "metric not found",
			initialState: nil,
			input:        models.Metrics{ID: "unknown"},
			wantErr:      ErrMetricNotFound,
			expectedID:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemStorage()
			if tt.initialState != nil {
				s.metrics = tt.initialState
			}

			// 1. Вызываем метод
			res, err := s.GetMetrica(context.Background(), tt.input)

			// 2. Проверяем ошибку через errors.Is или прямое сравнение
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got error %v, want %v", err, tt.wantErr)
			}

			// 3. Если ошибки нет, проверяем содержимое
			if tt.wantErr == nil {
				if res.ID != tt.expectedID {
					t.Errorf("got ID %s, want %s", res.ID, tt.expectedID)
				}
			}
		})
	}
}
func TestMemStorage_GetListMetrics_Table(t *testing.T) {
	floatPtr := func(f float64) *float64 { return &f }
	intPtr := func(i int64) *int64 { return &i }

	tests := []struct {
		name         string
		initialState map[string]models.Metrics
		expectedLen  int
	}{
		{
			name:         "empty storage",
			initialState: nil,
			expectedLen:  0,
		},
		{
			name: "multiple metrics",
			initialState: map[string]models.Metrics{
				"m1": {ID: "m1", MType: models.Gauge, Value: floatPtr(1.1)},
				"m2": {ID: "m2", MType: models.Counter, Delta: intPtr(10)},
			},
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemStorage()
			if tt.initialState != nil {
				s.metrics = tt.initialState
			}

			// 1. Вызываем метод
			res, err := s.GetListMetrics(context.Background())

			// 2. Проверяем на ошибки (в текущей реализации их быть не может, но для порядка)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// 3. Проверяем длину полученного слайса
			if len(res) != tt.expectedLen {
				t.Errorf("got slice length %d, want %d", len(res), tt.expectedLen)
			}

			// 4. (Опционально) Проверяем, что ID из результата есть в исходной мапе
			for _, m := range res {
				if _, ok := tt.initialState[m.ID]; !ok && tt.initialState != nil {
					t.Errorf("found unexpected metric ID in result: %s", m.ID)
				}
			}
		})
	}
}

func TestMetricsService_GetListMetrics_Integration(t *testing.T) {
	t.Run("empty storage returns empty slice", func(t *testing.T) {
		repo := NewMemStorage()
		srv := service.NewMetricsService(repo, nil)

		res, err := srv.GetListMetrics(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 0 {
			t.Errorf("expected 0 metrics, got %d", len(res))
		}
	})

	t.Run("returns all saved metrics", func(t *testing.T) {
		repo := NewMemStorage()
		srv := service.NewMetricsService(repo, nil)

		// 1. Сохраняем разные типы метрик
		_, _ = srv.ParseAndSave(context.Background(), models.Gauge, "g1", "1.1")
		_, _ = srv.ParseAndSave(context.Background(), models.Counter, "c1", "10")
		_, _ = srv.ParseAndSave(context.Background(), models.Gauge, "g2", "2.2")

		// 2. Получаем список
		res, err := srv.GetListMetrics(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 3. Проверяем количество
		if len(res) != 3 {
			t.Errorf("expected 3 metrics, got %d", len(res))
		}

		// 4. Проверяем наличие конкретных ID (порядок в map не гарантирован)
		foundIDs := make(map[string]bool)
		for _, m := range res {
			foundIDs[m.ID] = true
		}

		expectedIDs := []string{"g1", "c1", "g2"}
		for _, id := range expectedIDs {
			if !foundIDs[id] {
				t.Errorf("metric %s not found in result list", id)
			}
		}
	})

	t.Run("counter updates correctly in list", func(t *testing.T) {
		repo := NewMemStorage()
		srv := service.NewMetricsService(repo, nil)

		// Инкрементируем один и тот же счетчик дважды
		_, _ = srv.ParseAndSave(context.Background(), models.Counter, "c1", "10")
		_, _ = srv.ParseAndSave(context.Background(), models.Counter, "c1", "5")

		res, _ := srv.GetListMetrics(context.Background())

		if len(res) != 1 {
			t.Fatalf("expected 1 metric, got %d", len(res))
		}

		if *res[0].Delta != 15 {
			t.Errorf("expected delta 15, got %d", *res[0].Delta)
		}
	})
}

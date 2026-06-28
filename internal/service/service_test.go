package service

import (
	"context"
	"errors"
	"metrics/internal/model"
	"metrics/internal/repository"
	"testing"
)

// mockDB реализует интерфейс DB для тестирования сервиса в изоляции.
type mockDB struct {
	pingErr error
}

func (m *mockDB) Ping() error { return m.pingErr }

func TestMetricsService_ParseAndSave_Integration(t *testing.T) {

	strPtr := func(s string) *string { return &s }

	tests := []struct {
		wantErr       error
		setupValue    *string
		name          string
		mType         string
		id            string
		value         string
		expectedValue float64
		expectedDelta int64
	}{
		{
			name:          "new gauge",
			mType:         models.Gauge,
			id:            "g1",
			value:         "10.5",
			expectedValue: 10.5,
			wantErr:       nil,
		},
		{
			name:          "new counter",
			mType:         models.Counter,
			id:            "c2",
			value:         "10",
			expectedDelta: 10,
			wantErr:       nil,
		},
		{
			name:          "update gauge",
			mType:         models.Gauge,
			id:            "g1",
			setupValue:    strPtr("5.5"),
			value:         "10.5",
			expectedValue: 10.5,
			wantErr:       nil,
		},
		{
			name:          "increment counter",
			mType:         models.Counter,
			id:            "c1",
			setupValue:    strPtr("10"),
			value:         "5",
			expectedDelta: 15,
			wantErr:       nil,
		},
		{
			name:          "increment counter negative number",
			mType:         models.Counter,
			id:            "c1",
			setupValue:    strPtr("-10"),
			value:         "5",
			expectedDelta: -5,
			wantErr:       nil,
		},
		{
			name:    "invalid gauge value",
			mType:   models.Gauge,
			id:      "g2",
			value:   "bad_float",
			wantErr: ErrInvalidValue,
		},
		{
			name:    "invalid counter value",
			mType:   models.Counter,
			id:      "c2",
			value:   "10.5",
			wantErr: ErrInvalidValue,
		},
		{
			name:    "invalid type",
			mType:   "wrong",
			id:      "w1",
			value:   "10",
			wantErr: ErrTypeMetric,
		},
		{
			name:    "counter overflow int64",
			mType:   models.Counter,
			id:      "c_over",
			value:   "9223372036854775808",
			wantErr: ErrInvalidValue,
		},
		{
			name:    "counter underflow int64",
			mType:   models.Counter,
			id:      "c_under",
			value:   "-9223372036854775809",
			wantErr: ErrInvalidValue,
		},
		{
			name:    "gauge overflow float64",
			mType:   models.Gauge,
			id:      "g_over",
			value:   "1.7976931348623159e+309",
			wantErr: ErrInvalidValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemStorage()
			srv := NewMetricsService(repo, nil)

			if tt.setupValue != nil {
				_, err := srv.ParseAndSave(context.Background(), tt.mType, tt.id, *tt.setupValue)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			res, err := srv.ParseAndSave(context.Background(), tt.mType, tt.id, tt.value)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.mType == models.Gauge {
				if *res.Value != tt.expectedValue {
					t.Errorf("got value %v, want %v", *res.Value, tt.expectedValue)
				}
			} else if tt.mType == models.Counter {
				if *res.Delta != tt.expectedDelta {
					t.Errorf("got delta %v, want %v", *res.Delta, tt.expectedDelta)
				}
			}
		})
	}
}

func TestMetricsService_GetMetrica_Integration(t *testing.T) {
	repo := repository.NewMemStorage()
	srv := NewMetricsService(repo, nil)

	// Предзаполняем данными
	_, _ = srv.ParseAndSave(context.Background(), models.Gauge, "temp", "1.23")
	_, _ = srv.ParseAndSave(context.Background(), models.Counter, "poll", "5")

	t.Run("get existing gauge", func(t *testing.T) {
		m, err := srv.GetMetrica(context.Background(), models.Gauge, "temp")
		if err != nil || *m.Value != 1.23 {
			t.Errorf("failed to get gauge: %v", err)
		}
	})

	t.Run("get existing counter increment", func(t *testing.T) {
		// Добавляем еще 5 к существующему counter (5 + 5 = 10)
		_, _ = srv.ParseAndSave(context.Background(), models.Counter, "poll", "5")
		m, err := srv.GetMetrica(context.Background(), models.Counter, "poll")
		if err != nil || *m.Delta != 10 {
			t.Errorf("counter increment failed: %v", err)
		}
	})

	t.Run("get with wrong type", func(t *testing.T) {
		// Запрашиваем "temp" (который Gauge) как Counter
		_, err := srv.GetMetrica(context.Background(), models.Counter, "temp")
		if !errors.Is(err, ErrTypeMetric) {
			t.Errorf("expected ErrTypeMetric, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := srv.GetMetrica(context.Background(), models.Gauge, "non_existent")
		if !errors.Is(err, ErrMetricNotFound) {
			t.Errorf("expected ErrMetricNotFound, got %v", err)
		}
	})
}

func TestMetricsService_PingDB(t *testing.T) {
	repo := repository.NewMemStorage()

	tests := []struct {
		db      DB
		wantErr error
		name    string
	}{
		{
			name:    "БД доступна",
			db:      &mockDB{pingErr: nil},
			wantErr: nil,
		},
		{
			name:    "БД недоступна",
			db:      &mockDB{pingErr: errors.New("connection refused")},
			wantErr: errors.New("connection refused"),
		},
		{
			name:    "БД не инициализирована",
			db:      nil,
			wantErr: ErrDBNotInit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMetricsService(repo, tt.db)
			err := svc.PingDB()

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if tt.db == nil {
				if !errors.Is(err, ErrDBNotInit) {
					t.Errorf("expected ErrDBNotInit, got %v", err)
				}
				return
			}

			if err.Error() != tt.wantErr.Error() {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestMetricsService_GetListMetrics(t *testing.T) {
	repo := repository.NewMemStorage()
	svc := NewMetricsService(repo, nil)

	t.Run("пустое хранилище", func(t *testing.T) {
		list, err := svc.GetListMetrics(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("expected empty list, got %d items", len(list))
		}
	})

	t.Run("возвращает все метрики", func(t *testing.T) {
		v := 1.5
		_, _ = repo.UpdateGauges(context.Background(), models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &v})

		list, err := svc.GetListMetrics(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) < 1 {
			t.Errorf("expected at least 1 metric, got %d", len(list))
		}
	})
}

func TestMetricsService_UpdateBatch(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }
	intPtr := func(v int64) *int64 { return &v }

	t.Run("батч сохраняется", func(t *testing.T) {
		repo := repository.NewMemStorage()
		svc := NewMetricsService(repo, nil)

		metrics := []models.Metrics{
			{ID: "Alloc", MType: models.Gauge, Value: floatPtr(1.5)},
			{ID: "PollCount", MType: models.Counter, Delta: intPtr(5)},
		}

		err := svc.UpdateBatch(context.Background(), metrics)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, err := svc.GetMetrica(context.Background(), models.Gauge, "Alloc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *m.Value != 1.5 {
			t.Errorf("expected 1.5, got %v", *m.Value)
		}
	})
}

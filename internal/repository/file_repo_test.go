package repository

import (
	"encoding/json"
	models "metrics/internal/model"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileBackedRepo_Load(t *testing.T) {
	floatPtr := func(f float64) *float64 { return &f }
	intPtr := func(i int64) *int64 { return &i }

	tests := []struct {
		name        string
		fileContent string // пустая строка = файл не создаётся
		wantErr     bool
		wantMetrics []models.Metrics
	}{
		{
			name:        "file not exists",
			fileContent: "",
			wantErr:     false,
			wantMetrics: nil,
		},
		{
			name:        "invalid JSON",
			fileContent: "not a json",
			wantErr:     true,
			wantMetrics: nil,
		},
		{
			name:        "empty array",
			fileContent: "[]",
			wantErr:     false,
			wantMetrics: []models.Metrics{},
		},
		{
			name: "load gauge",
			fileContent: func() string {
				m := []models.Metrics{{ID: "g1", MType: models.Gauge, Value: floatPtr(1.5)}}
				b, _ := json.Marshal(m)
				return string(b)
			}(),
			wantErr:     false,
			wantMetrics: []models.Metrics{{ID: "g1", MType: models.Gauge, Value: floatPtr(1.5)}},
		},
		{
			name: "load counter",
			fileContent: func() string {
				m := []models.Metrics{{ID: "c1", MType: models.Counter, Delta: intPtr(10)}}
				b, _ := json.Marshal(m)
				return string(b)
			}(),
			wantErr:     false,
			wantMetrics: []models.Metrics{{ID: "c1", MType: models.Counter, Delta: intPtr(10)}},
		},
		{
			name: "load multiple metrics",
			fileContent: func() string {
				m := []models.Metrics{
					{ID: "g1", MType: models.Gauge, Value: floatPtr(2.2)},
					{ID: "c1", MType: models.Counter, Delta: intPtr(5)},
				}
				b, _ := json.Marshal(m)
				return string(b)
			}(),
			wantErr: false,
			wantMetrics: []models.Metrics{
				{ID: "g1", MType: models.Gauge, Value: floatPtr(2.2)},
				{ID: "c1", MType: models.Counter, Delta: intPtr(5)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			fp := filepath.Join(dir, "storage.txt")

			if tt.fileContent != "" {
				os.WriteFile(fp, []byte(tt.fileContent), 0666)
			}

			mem := NewMemStorage()
			repo := NewFileBackedRepo(mem, fp, 300)

			err := repo.load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			list, _ := mem.GetListMetrics()
			if len(list) != len(tt.wantMetrics) {
				t.Fatalf("got %d metrics, want %d", len(list), len(tt.wantMetrics))
			}

			for _, want := range tt.wantMetrics {
				got, err := mem.GetMetrica(want)
				if err != nil {
					t.Errorf("metric %s not found after Load", want.ID)
					continue
				}
				if got.MType != want.MType {
					t.Errorf("metric %s: got type %s, want %s", want.ID, got.MType, want.MType)
				}
			}
		})
	}
}

func TestFileBackedRepo_Save(t *testing.T) {
	floatPtr := func(f float64) *float64 { return &f }
	intPtr := func(i int64) *int64 { return &i }

	tests := []struct {
		name         string
		initialState map[string]models.Metrics
		wantLen      int
	}{
		{
			name:         "save empty storage",
			initialState: nil,
			wantLen:      0,
		},
		{
			name: "save gauge",
			initialState: map[string]models.Metrics{
				"g1": {ID: "g1", MType: models.Gauge, Value: floatPtr(9.9)},
			},
			wantLen: 1,
		},
		{
			name: "save multiple metrics",
			initialState: map[string]models.Metrics{
				"g1": {ID: "g1", MType: models.Gauge, Value: floatPtr(1.1)},
				"c1": {ID: "c1", MType: models.Counter, Delta: intPtr(42)},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			fp := filepath.Join(dir, "storage.txt")

			mem := NewMemStorage()
			if tt.initialState != nil {
				mem.metrics = tt.initialState
			}
			repo := NewFileBackedRepo(mem, fp, 300)

			if err := repo.Save(); err != nil {
				t.Fatalf("Save() unexpected error: %v", err)
			}

			data, err := os.ReadFile(fp)
			if err != nil {
				t.Fatalf("file not created: %v", err)
			}

			var saved []models.Metrics
			if err := json.Unmarshal(data, &saved); err != nil {
				t.Fatalf("invalid JSON in file: %v", err)
			}

			if len(saved) != tt.wantLen {
				t.Errorf("got %d metrics in file, want %d", len(saved), tt.wantLen)
			}
		})
	}
}

func TestFileBackedRepo_ImmediateSave(t *testing.T) {
	floatPtr := func(f float64) *float64 { return &f }
	intPtr := func(i int64) *int64 { return &i }

	tests := []struct {
		name     string
		interval time.Duration
		metric   models.Metrics
		wantSave bool // ожидаем ли запись в файл после update
	}{
		{
			name:     "gauge: interval=0 saves immediately",
			interval: 0,
			metric:   models.Metrics{ID: "g1", MType: models.Gauge, Value: floatPtr(5.5)},
			wantSave: true,
		},
		{
			name:     "gauge: interval>0 does not save",
			interval: time.Hour,
			metric:   models.Metrics{ID: "g1", MType: models.Gauge, Value: floatPtr(5.5)},
			wantSave: false,
		},
		{
			name:     "counter: interval=0 saves immediately",
			interval: 0,
			metric:   models.Metrics{ID: "c1", MType: models.Counter, Delta: intPtr(3)},
			wantSave: true,
		},
		{
			name:     "counter: interval>0 does not save",
			interval: time.Hour,
			metric:   models.Metrics{ID: "c1", MType: models.Counter, Delta: intPtr(3)},
			wantSave: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			fp := filepath.Join(dir, "storage.txt")

			mem := NewMemStorage()
			repo := &FileBackedRepo{mem: mem, filePath: fp, interval: tt.interval}

			switch tt.metric.MType {
			case models.Gauge:
				repo.UpdateGauges(tt.metric)
			case models.Counter:
				repo.UpdateCounter(tt.metric)
			}

			_, err := os.Stat(fp)
			fileExists := err == nil

			if fileExists != tt.wantSave {
				t.Errorf("file exists = %v, wantSave = %v", fileExists, tt.wantSave)
			}
		})
	}
}

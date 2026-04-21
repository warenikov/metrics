package models

import "testing"

func TestMetrics_ValueString(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }
	int64Ptr := func(v int64) *int64 { return &v }

	tests := []struct {
		name     string
		metric   Metrics
		expected string
	}{
		{
			name:     "gauge с value",
			metric:   Metrics{MType: Gauge, Value: floatPtr(100.5)},
			expected: "100.5",
		},
		{
			name:     "gauge с нулевым value",
			metric:   Metrics{MType: Gauge, Value: floatPtr(0)},
			expected: "0",
		},
		{
			name:     "gauge с nil value",
			metric:   Metrics{MType: Gauge},
			expected: "0",
		},
		{
			name:     "counter с delta",
			metric:   Metrics{MType: Counter, Delta: int64Ptr(42)},
			expected: "42",
		},
		{
			name:     "counter с отрицательной delta",
			metric:   Metrics{MType: Counter, Delta: int64Ptr(-10)},
			expected: "-10",
		},
		{
			name:     "counter с nil delta",
			metric:   Metrics{MType: Counter},
			expected: "0",
		},
		{
			name:     "неизвестный тип",
			metric:   Metrics{MType: "unknown"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metric.ValueString()
			if got != tt.expected {
				t.Errorf("ValueString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

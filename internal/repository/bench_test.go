package repository

import (
	"context"
	"fmt"
	"testing"

	models "metrics/internal/model"
)

func BenchmarkMemStorage_UpdateGauge(b *testing.B) {
	b.ReportAllocs()
	store := NewMemStorage()
	v := 1.23
	m := models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &v}
	for b.Loop() {
		store.UpdateGauges(context.Background(), m)
	}
}

func BenchmarkMemStorage_UpdateCounter(b *testing.B) {
	b.ReportAllocs()
	store := NewMemStorage()
	d := int64(1)
	m := models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &d}
	for b.Loop() {
		store.UpdateCounter(context.Background(), m)
	}
}

func BenchmarkMemStorage_GetListMetrics(b *testing.B) {
	b.ReportAllocs()
	store := NewMemStorage()
	for i := range 100 {
		v := float64(i)
		store.UpdateGauges(context.Background(), models.Metrics{
			ID: fmt.Sprintf("metric_%d", i), MType: models.Gauge, Value: &v,
		})
	}
	for b.Loop() {
		store.GetListMetrics(context.Background())
	}
}

func BenchmarkMemStorage_UpdateBatch(b *testing.B) {
	b.ReportAllocs()
	store := NewMemStorage()

	batch := make([]models.Metrics, 30)
	for i := range batch {
		v := float64(i)
		batch[i] = models.Metrics{ID: fmt.Sprintf("m%d", i), MType: models.Gauge, Value: &v}
	}

	for b.Loop() {
		store.UpdateBatch(context.Background(), batch)
	}
}

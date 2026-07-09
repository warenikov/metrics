package service

import (
	"context"
	"fmt"
	"testing"

	models "metrics/internal/model"
	"metrics/internal/repository"
)

func BenchmarkService_ParseAndSave_Gauge(b *testing.B) {
	b.ReportAllocs()
	svc := NewMetricsService(repository.NewMemStorage(), nil)
	for b.Loop() {
		_, _ = svc.ParseAndSave(context.Background(), models.Gauge, "Alloc", "1.234")
	}
}

func BenchmarkService_ParseAndSave_Counter(b *testing.B) {
	b.ReportAllocs()
	svc := NewMetricsService(repository.NewMemStorage(), nil)
	for b.Loop() {
		_, _ = svc.ParseAndSave(context.Background(), models.Counter, "PollCount", "1")
	}
}

func BenchmarkService_UpdateBatch(b *testing.B) {
	b.ReportAllocs()
	svc := NewMetricsService(repository.NewMemStorage(), nil)

	batch := make([]models.Metrics, 30)
	for i := range batch {
		v := float64(i)
		batch[i] = models.Metrics{ID: fmt.Sprintf("m%d", i), MType: models.Gauge, Value: &v}
	}

	for b.Loop() {
		_ = svc.UpdateBatch(context.Background(), batch)
	}
}

func BenchmarkService_GetListMetrics(b *testing.B) {
	b.ReportAllocs()
	repo := repository.NewMemStorage()
	svc := NewMetricsService(repo, nil)

	for i := range 50 {
		v := float64(i)
		_, _ = repo.UpdateGauges(context.Background(), models.Metrics{
			ID: fmt.Sprintf("m%d", i), MType: models.Gauge, Value: &v,
		})
	}

	for b.Loop() {
		_, _ = svc.GetListMetrics(context.Background())
	}
}

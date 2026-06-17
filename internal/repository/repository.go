// Package repository provides in-memory and file-backed implementations of the
// metrics store used by the service layer.
package repository

import (
	"context"
	"errors"
	"metrics/internal/model"
)

// Sentinel errors returned by MemStorage operations.
var (
	// ErrInvalidValue is returned when a required metric field (such as Delta) is nil.
	ErrInvalidValue = errors.New("invalid metric value")
	// ErrMetricNotFound is returned when a requested metric does not exist in the store.
	ErrMetricNotFound = errors.New("metric not found")
)

// MemStorage is a thread-unsafe in-memory metrics store keyed by metric ID.
// It is intended for use behind a single-threaded service or a synchronised wrapper.
type MemStorage struct {
	metrics map[string]models.Metrics
}

// NewMemStorage creates an empty MemStorage.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}

// UpdateGauges stores the gauge metric m, replacing any existing value with the same ID.
func (s *MemStorage) UpdateGauges(_ context.Context, m models.Metrics) (models.Metrics, error) {
	s.metrics[m.ID] = m
	return m, nil
}

// UpdateCounter adds m.Delta to the existing counter value for m.ID and stores the result.
// Returns ErrInvalidValue if m.Delta is nil.
func (s *MemStorage) UpdateCounter(_ context.Context, m models.Metrics) (models.Metrics, error) {
	if m.Delta == nil {
		return m, ErrInvalidValue
	}

	newVal := *m.Delta
	if old, ok := s.metrics[m.ID]; ok && old.Delta != nil {
		newVal += *old.Delta
	}
	m.Delta = &newVal
	s.metrics[m.ID] = m
	return m, nil
}

// GetMetrica returns the metric identified by m.ID, or ErrMetricNotFound if absent.
func (s *MemStorage) GetMetrica(_ context.Context, m models.Metrics) (*models.Metrics, error) {
	mm, ok := s.metrics[m.ID]
	if !ok {
		return nil, ErrMetricNotFound
	}
	return &mm, nil
}

// UpdateBatch applies gauge replacements and counter increments for every metric in the slice.
// Returns ErrInvalidValue on the first counter metric whose Delta is nil.
func (s *MemStorage) UpdateBatch(_ context.Context, metrics []models.Metrics) error {
	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			s.metrics[m.ID] = m
		case models.Counter:
			if m.Delta == nil {
				return ErrInvalidValue
			}
			newVal := *m.Delta
			if old, ok := s.metrics[m.ID]; ok && old.Delta != nil {
				newVal += *old.Delta
			}
			m.Delta = &newVal
			s.metrics[m.ID] = m
		}
	}
	return nil
}

// GetListMetrics returns a snapshot of all stored metrics as a slice.
func (s *MemStorage) GetListMetrics(_ context.Context) ([]models.Metrics, error) {
	res := make([]models.Metrics, 0, len(s.metrics))
	for _, m := range s.metrics {
		res = append(res, m)
	}
	return res, nil
}

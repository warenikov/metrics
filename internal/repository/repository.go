package storage

import (
	"metrics/internal/model"
)

type Repository interface {
	UpdateCauges(m models.Metrics) (models.Metrics, error)
	UpdateCounter(m models.Metrics) (models.Metrics, error)
}

type MemStorage struct {
	metrics map[string]models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}

func (s *MemStorage) UpdateCauges(m models.Metrics) (models.Metrics, error) {
	s.metrics[m.ID] = m
	return m, nil
}

func (s *MemStorage) UpdateCounter(m models.Metrics) (models.Metrics, error) {
	newVal := *m.Delta
	if old, ok := s.metrics[m.ID]; ok && old.Delta != nil {
		newVal += *old.Delta
	}
	m.Delta = &newVal
	s.metrics[m.ID] = m
	return m, nil
}

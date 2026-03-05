package storage

import (
	"metrics/internal/model"
)

type MemStorage struct {
	metrics map[string]models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}

func (s *MemStorage) UpdateGauges(m models.Metrics) (models.Metrics, error) {
	s.metrics[m.ID] = m
	return m, nil
}

func (s *MemStorage) UpdateCounter(m models.Metrics) (models.Metrics, error) {
	if m.Delta == nil {
		return m, models.ErrInvalidValue
	}

	newVal := *m.Delta
	if old, ok := s.metrics[m.ID]; ok && old.Delta != nil {
		newVal += *old.Delta
	}
	m.Delta = &newVal
	s.metrics[m.ID] = m
	return m, nil
}

func (s *MemStorage) GetMetrica(m models.Metrics) (*models.Metrics, error) {
	mm, ok := s.metrics[m.ID]
	if !ok {
		return nil, models.ErrMetricNotFound
	}
	return &mm, nil
}

func (s *MemStorage) GetListMetrics() ([]models.Metrics, error) {
	res := make([]models.Metrics, 0, len(s.metrics))
	for _, m := range s.metrics {
		res = append(res, m)
	}
	return res, nil
}

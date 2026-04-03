package service

import (
	"errors"
	"fmt"
	"metrics/internal/model"
	"strconv"
)

type MetricsService struct {
	repo Repository
	db   DB
}

var (
	ErrInvalidValue   = errors.New("invalid metric value")
	ErrMetricNotFound = errors.New("metric not found")
	ErrTypeMetric     = errors.New("invalid metric type")
	ErrDBNotInit      = errors.New("database not initialized")
)

type Repository interface {
	UpdateGauges(m models.Metrics) (models.Metrics, error)
	UpdateCounter(m models.Metrics) (models.Metrics, error)
	GetMetrica(m models.Metrics) (*models.Metrics, error)
	GetListMetrics() ([]models.Metrics, error)
}

type DB interface {
	Ping() error
}

// NewMetricsService — конструктор, принимающий интерфейс репозитория
func NewMetricsService(r Repository, db DB) *MetricsService {
	return &MetricsService{repo: r, db: db}
}

func (s *MetricsService) GetListMetrics() ([]models.Metrics, error) {
	return s.repo.GetListMetrics()
}

func (s *MetricsService) GetMetrica(mType, id string) (*models.Metrics, error) {
	query := models.Metrics{
		ID:    id,
		MType: mType,
	}

	res, err := s.repo.GetMetrica(query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetricNotFound, err)
	}

	if res.MType != mType {
		return nil, ErrTypeMetric
	}

	return res, nil
}

// ParseAndSave берет сырые строки из хендлера, превращает в модель и отдает в репо
func (s *MetricsService) ParseAndSave(mType, id, value string) (models.Metrics, error) {
	metric := models.Metrics{ID: id, MType: mType}

	switch mType {
	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return metric, fmt.Errorf("%w: %v", ErrInvalidValue, err)
		}
		metric.Value = &v

		saved, err := s.repo.UpdateGauges(metric)
		if err != nil {
			return saved, fmt.Errorf("error update gauge: %w:%v", ErrInvalidValue, err)
		}
		return saved, nil

	case models.Counter:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return metric, fmt.Errorf("%w: %v", ErrInvalidValue, err)
		}
		metric.Delta = &v

		saved, err := s.repo.UpdateCounter(metric)
		if err != nil {
			return saved, fmt.Errorf("error update counter: %w:%v", ErrInvalidValue, err)
		}
		return saved, nil

	default:
		return metric, ErrTypeMetric
	}
}

func (s *MetricsService) PingDB() error {
	if s.db == nil {
		return ErrDBNotInit
	}

	return s.db.Ping()
}

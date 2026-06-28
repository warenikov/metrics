// Package service implements the business logic layer of the metrics server.
// It sits between the HTTP handlers and the storage repository, enforcing
// type validation and value parsing before delegation to the repository.
package service

import (
	"context"
	"errors"
	"fmt"
	"metrics/internal/model"
	"strconv"
)

// MetricsService implements the core metrics operations using a Repository backend.
type MetricsService struct {
	repo Repository
	db   DB
}

// Sentinel errors returned by MetricsService methods.
var (
	// ErrInvalidValue is returned when a metric value cannot be parsed.
	ErrInvalidValue = errors.New("invalid metric value")
	// ErrMetricNotFound is returned when a requested metric does not exist in the store.
	ErrMetricNotFound = errors.New("metric not found")
	// ErrTypeMetric is returned when the metric type is not recognised.
	ErrTypeMetric = errors.New("invalid metric type")
	// ErrDBNotInit is returned when a DB operation is requested but no database is configured.
	ErrDBNotInit = errors.New("database not initialized")
)

// Repository is the storage interface required by MetricsService.
type Repository interface {
	// UpdateGauges stores a gauge metric, replacing any previous value.
	UpdateGauges(ctx context.Context, m models.Metrics) (models.Metrics, error)
	// UpdateCounter increments the counter metric by the delta in m, persisting the cumulative value.
	UpdateCounter(ctx context.Context, m models.Metrics) (models.Metrics, error)
	// GetMetrica retrieves a metric by ID and type from the store.
	GetMetrica(ctx context.Context, m models.Metrics) (*models.Metrics, error)
	// GetListMetrics returns all stored metrics.
	GetListMetrics(ctx context.Context) ([]models.Metrics, error)
	// UpdateBatch atomically stores a slice of metrics, applying gauge replacement and counter accumulation.
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
}

// DB is the database health interface used for the /ping endpoint.
type DB interface {
	// Ping verifies that the database connection is alive.
	Ping() error
}

// NewMetricsService constructs a MetricsService backed by the given Repository and DB.
// db may be nil if no database is configured; in that case PingDB returns ErrDBNotInit.
func NewMetricsService(r Repository, db DB) *MetricsService {
	return &MetricsService{repo: r, db: db}
}

// GetListMetrics returns all metrics currently stored in the repository.
func (s *MetricsService) GetListMetrics(ctx context.Context) ([]models.Metrics, error) {
	return s.repo.GetListMetrics(ctx)
}

// GetMetrica retrieves a single metric by type and id.
// Returns ErrMetricNotFound if the metric does not exist,
// or ErrTypeMetric if the stored type differs from the requested type.
func (s *MetricsService) GetMetrica(ctx context.Context, mType, id string) (*models.Metrics, error) {
	query := models.Metrics{
		ID:    id,
		MType: mType,
	}

	res, err := s.repo.GetMetrica(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetricNotFound, err)
	}

	if res.MType != mType {
		return nil, ErrTypeMetric
	}

	return res, nil
}

// ParseAndSave parses the raw string value, converts it to the appropriate numeric type,
// and delegates to the repository. Returns ErrInvalidValue for unparseable values
// and ErrTypeMetric for unknown metric types.
func (s *MetricsService) ParseAndSave(ctx context.Context, mType, id, value string) (models.Metrics, error) {
	metric := models.Metrics{ID: id, MType: mType}

	switch mType {
	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return metric, fmt.Errorf("%w: %v", ErrInvalidValue, err)
		}
		metric.Value = &v

		saved, err := s.repo.UpdateGauges(ctx, metric)
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

		saved, err := s.repo.UpdateCounter(ctx, metric)
		if err != nil {
			return saved, fmt.Errorf("error update counter: %w:%v", ErrInvalidValue, err)
		}
		return saved, nil

	default:
		return metric, ErrTypeMetric
	}
}

// UpdateBatch stores a slice of metrics in a single operation.
func (s *MetricsService) UpdateBatch(ctx context.Context, metrics []models.Metrics) error {
	return s.repo.UpdateBatch(ctx, metrics)
}

// PingDB checks whether the database is reachable.
// Returns ErrDBNotInit if no database was provided at construction time.
func (s *MetricsService) PingDB() error {
	if s.db == nil {
		return ErrDBNotInit
	}

	return s.db.Ping()
}

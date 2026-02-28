package service

import (
	"metrics/internal/model"
	"metrics/internal/repository"
	"strconv"
)

type Service interface {
	ParseAndSave(mType, id, value string) (models.Metrics, error)
	GetMetrica(mType, id string) (*models.Metrics, error)
	GetListMetrics() ([]models.Metrics, error)
}

type MetricsService struct {
	repo storage.Repository
}

func (s *MetricsService) GetListMetrics() ([]models.Metrics, error) {
	return s.repo.GetListMetrics()
}

// NewMetricsService — конструктор, принимающий интерфейс репозитория
func NewMetricsService(r storage.Repository) *MetricsService {
	return &MetricsService{repo: r}
}

func (s *MetricsService) GetMetrica(mType, id string) (*models.Metrics, error) {
	query := models.Metrics{
		ID:    id,
		MType: mType,
	}

	res, err := s.repo.GetMetrica(query)
	if err != nil {
		//TODO: логирование + обратка ошибок
		return nil, err
	}

	if res.MType != mType {
		//TODO: логирование + обратка ошибок
		return nil, models.ErrMetricNotFound
	}

	return res, nil
}

// ParseAndSave берет сырые строки из хендлера, превращает в модель и отдает в репо
func (s *MetricsService) ParseAndSave(mType, id, value string) (models.Metrics, error) {
	metric := models.Metrics{
		ID:    id,
		MType: mType,
	}

	switch mType {
	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			//TODO: логирование + обратка ошибок
			return metric, models.ErrInvalidValue
		}
		metric.Value = &v

		return s.repo.UpdateGauges(metric)

	case models.Counter:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			//TODO: логирование + обратка ошибок
			return metric, models.ErrInvalidValue
		}
		metric.Delta = &v

		return s.repo.UpdateCounter(metric)

	default:
		//TODO: логирование + обратка ошибок
		return metric, models.ErrInvalidMetricType
	}

}

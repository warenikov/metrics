package repository

import (
	"encoding/json"
	"metrics/internal/logger"
	models "metrics/internal/model"
	"os"
	"time"

	"go.uber.org/zap"
)

const filePermissions = 0666

type FileBackedRepo struct {
	mem      *MemStorage
	filePath string
	interval time.Duration
}

func NewFileBackedRepo(mem *MemStorage, filePath string, intervalSec int) *FileBackedRepo {
	return &FileBackedRepo{
		mem:      mem,
		filePath: filePath,
		interval: time.Duration(intervalSec) * time.Second,
	}
}

func (r *FileBackedRepo) UpdateGauges(m models.Metrics) (models.Metrics, error) {
	result, err := r.mem.UpdateGauges(m)
	if err == nil && r.interval == 0 {
		r.saveToFile()
	}
	return result, err
}

func (r *FileBackedRepo) UpdateCounter(m models.Metrics) (models.Metrics, error) {
	result, err := r.mem.UpdateCounter(m)
	if err == nil && r.interval == 0 {
		r.saveToFile()
	}
	return result, err
}

func (r *FileBackedRepo) GetMetrica(m models.Metrics) (*models.Metrics, error) {
	return r.mem.GetMetrica(m)
}

func (r *FileBackedRepo) GetListMetrics() ([]models.Metrics, error) {
	return r.mem.GetListMetrics()
}

func (r *FileBackedRepo) Load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			_, e := r.mem.UpdateGauges(m)
			if e != nil {
				return e
			}
		case models.Counter:
			_, e := r.mem.UpdateCounter(m)
			if e != nil {
				return e
			}
		}
	}
	return nil
}

func (r *FileBackedRepo) RunSave() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for range ticker.C {
		r.saveToFile()
	}
}

func (r *FileBackedRepo) Save() error {
	metrics, err := r.mem.GetListMetrics()
	if err != nil {
		return err
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, filePermissions)
}

func (r *FileBackedRepo) saveToFile() {
	metrics, err := r.mem.GetListMetrics()
	if err != nil {
		logger.Log.Error("Error getting metrics for save", zap.Error(err))
		return
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		logger.Log.Error("Error marshaling metrics", zap.Error(err))
		return
	}
	if err := os.WriteFile(r.filePath, data, filePermissions); err != nil {
		logger.Log.Error("Error writing metrics to file", zap.Error(err))
	}
}

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
	mem            *MemStorage
	filePath       string
	interval       time.Duration
	file           *os.File
	SyncDumpToFile bool
}

func NewFileBackedRepo(mem *MemStorage, filePath string, intervalSec uint, restore bool) (*FileBackedRepo, error) {

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, filePermissions)
	if err != nil {
		logger.Log.Error("Failed to open file", zap.String("filePath", filePath), zap.Error(err))
		return nil, err
	}

	repo := &FileBackedRepo{
		mem:            mem,
		filePath:       filePath,
		interval:       time.Duration(intervalSec) * time.Second,
		file:           f,
		SyncDumpToFile: intervalSec == 0,
	}

	if restore {
		if err = repo.load(); err != nil {
			logger.Log.Error("Error loading file backed repo", zap.Error(err))
			return nil, err
		}
	}

	return repo, nil
}

func (r *FileBackedRepo) UpdateGauges(m models.Metrics) (models.Metrics, error) {
	result, err := r.mem.UpdateGauges(m)
	if err == nil && r.SyncDumpToFile {
		if err := r.saveToFile(); err != nil {
			logger.Log.Error("Failed to save metrics to file", zap.Error(err))
		}
	}
	return result, err
}

func (r *FileBackedRepo) UpdateCounter(m models.Metrics) (models.Metrics, error) {
	result, err := r.mem.UpdateCounter(m)
	if err == nil && r.SyncDumpToFile {
		if err := r.saveToFile(); err != nil {
			logger.Log.Error("Failed to save metrics to file", zap.Error(err))
		}
	}
	return result, err
}

func (r *FileBackedRepo) GetMetrica(m models.Metrics) (*models.Metrics, error) {
	return r.mem.GetMetrica(m)
}

func (r *FileBackedRepo) GetListMetrics() ([]models.Metrics, error) {
	return r.mem.GetListMetrics()
}

func (r *FileBackedRepo) load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
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
		if err := r.saveToFile(); err != nil {
			logger.Log.Error("Failed to save metrics to file", zap.Error(err))
		}
	}
}

func (r *FileBackedRepo) Save() error {
	return r.saveToFile()
}

func (r *FileBackedRepo) saveToFile() error {
	metrics, err := r.mem.GetListMetrics()
	if err != nil {
		logger.Log.Error("Error getting metrics for save", zap.Error(err))
		return err
	}

	if err := r.file.Truncate(0); err != nil {
		return err
	}
	if _, err := r.file.Seek(0, 0); err != nil {
		return err
	}

	encoder := json.NewEncoder(r.file)
	if err = encoder.Encode(metrics); err != nil {
		logger.Log.Error("Error encoding metrics to file", zap.Error(err))
		return err
	}

	return nil
}

func (r *FileBackedRepo) Close() error {
	return r.file.Close()
}

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"metrics/internal/logger"
	models "metrics/internal/model"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

const filePermissions = 0666

type FileBackedRepo struct {
	mem            *MemStorage
	file           *os.File
	filePath       string
	interval       time.Duration
	fileMu         sync.Mutex
	SyncDumpToFile bool
}

func NewFileBackedRepo(mem *MemStorage, filePath string, intervalSec uint, restore bool) (*FileBackedRepo, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, filePermissions)
	if err != nil {
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
			return nil, err
		}
	}

	return repo, nil
}

func (r *FileBackedRepo) UpdateGauges(ctx context.Context, m models.Metrics) (models.Metrics, error) {
	result, err := r.mem.UpdateGauges(ctx, m)
	if err != nil {
		return result, err
	}
	if r.SyncDumpToFile {
		if err := r.saveToFile(); err != nil {
			return result, fmt.Errorf("sync save failed: %w", err)
		}
	}
	return result, nil
}

func (r *FileBackedRepo) UpdateCounter(ctx context.Context, m models.Metrics) (models.Metrics, error) {
	result, err := r.mem.UpdateCounter(ctx, m)
	if err != nil {
		return result, err
	}
	if r.SyncDumpToFile {
		if err := r.saveToFile(); err != nil {
			return result, fmt.Errorf("sync save failed: %w", err)
		}
	}
	return result, nil
}

func (r *FileBackedRepo) UpdateBatch(ctx context.Context, metrics []models.Metrics) error {
	if err := r.mem.UpdateBatch(ctx, metrics); err != nil {
		return err
	}
	if r.SyncDumpToFile {
		return r.saveToFile()
	}
	return nil
}

func (r *FileBackedRepo) GetMetrica(ctx context.Context, m models.Metrics) (*models.Metrics, error) {
	return r.mem.GetMetrica(ctx, m)
}

func (r *FileBackedRepo) GetListMetrics(ctx context.Context) ([]models.Metrics, error) {
	return r.mem.GetListMetrics(ctx)
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
			_, e := r.mem.UpdateGauges(context.Background(), m)
			if e != nil {
				return e
			}
		case models.Counter:
			_, e := r.mem.UpdateCounter(context.Background(), m)
			if e != nil {
				return e
			}
		}
	}
	return nil
}

// RunSave periodically saves metrics to file until ctx is cancelled. Callers
// should cancel ctx before making a final, definitive Save() call on
// shutdown, so the two never race over the file.
func (r *FileBackedRepo) RunSave(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.saveToFile(); err != nil {
				logger.Log.Error("Failed to save metrics to file", zap.Error(err))
			}
		}
	}
}

func (r *FileBackedRepo) Save() error {
	return r.saveToFile()
}

// saveToFile overwrites the backing file with the current in-memory metrics.
// fileMu serializes callers (periodic RunSave, synchronous per-request saves,
// and the final shutdown Save()) so their read-snapshot-then-write sequences
// never interleave: taking the snapshot under the same lock as the write
// guarantees whichever call acquires fileMu last also writes the freshest
// data, instead of possibly overwriting a newer save with a stale snapshot
// it read before waiting on the lock.
func (r *FileBackedRepo) saveToFile() error {
	r.fileMu.Lock()
	defer r.fileMu.Unlock()

	metrics, err := r.mem.GetListMetrics(context.Background())
	if err != nil {
		return err
	}

	if err = r.file.Truncate(0); err != nil {
		return err
	}
	if _, err = r.file.Seek(0, 0); err != nil {
		return err
	}

	encoder := json.NewEncoder(r.file)
	if err = encoder.Encode(metrics); err != nil {
		return err
	}

	return nil
}

func (r *FileBackedRepo) Close() error {
	r.fileMu.Lock()
	defer r.fileMu.Unlock()
	return r.file.Close()
}

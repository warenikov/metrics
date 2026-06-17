package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// FileObserver записывает события аудита в файл (по одному JSON на строку).
type FileObserver struct {
	f *os.File
}

// NewFileObserver открывает файл в режиме append и возвращает наблюдатель.
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("audit file open: %w", err)
	}
	return &FileObserver{f: f}, nil
}

func (o *FileObserver) Notify(_ context.Context, event AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit marshal: %w", err)
	}
	data = append(data, '\n')
	if _, err := o.f.Write(data); err != nil {
		return fmt.Errorf("audit file write: %w", err)
	}
	return nil
}

// Close закрывает файл аудита.
func (o *FileObserver) Close() error {
	return o.f.Close()
}

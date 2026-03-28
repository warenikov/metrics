package compress

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// Compress сжимает данные методом Gzip.
func Compress(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	return &buf, nil
}

// NewReader создаёт gzip-reader для разжатия входящих данных.
func NewReader(r io.Reader) (*gzip.Reader, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	return gr, nil
}

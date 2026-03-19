package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html") {
		w.Header().Set("Content-Encoding", "gzip")
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html") {
		w.Header().Set("Content-Encoding", "gzip")
		return w.Writer.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer gzReader.Close()
			r.Body = gzReader
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gzWriter := gzip.NewWriter(w)
		defer gzWriter.Close()

		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gzWriter,
		}

		next.ServeHTTP(gzw, r)
	})
}

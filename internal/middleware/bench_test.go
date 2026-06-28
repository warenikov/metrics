package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func benchGzip(b *testing.B, compressed bool) {
	b.Helper()
	b.ReportAllocs()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"Alloc","type":"gauge","value":1.5}`)) //nolint:errcheck
	})
	h := GzipMiddleware(handler)

	for b.Loop() {
		b.StopTimer()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if compressed {
			req.Header.Set("Accept-Encoding", "gzip")
		}
		w := httptest.NewRecorder()
		b.StartTimer()

		h.ServeHTTP(w, req)
	}
}

func BenchmarkGzipMiddleware_WithGzip(b *testing.B) {
	benchGzip(b, true)
}

func BenchmarkGzipMiddleware_NoGzip(b *testing.B) {
	benchGzip(b, false)
}

func benchLogger(b *testing.B) {
	b.Helper()
	b.ReportAllocs()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK")) //nolint:errcheck
	})
	h := LoggerMiddleware(handler)

	for b.Loop() {
		b.StopTimer()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		b.StartTimer()

		h.ServeHTTP(w, req)
	}
}

func BenchmarkLoggerMiddleware(b *testing.B) {
	benchLogger(b)
}

func benchHash(b *testing.B, key string) {
	b.Helper()
	b.ReportAllocs()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK")) //nolint:errcheck
	})
	h := HashMiddleware(key)(handler)

	for b.Loop() {
		b.StopTimer()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		b.StartTimer()

		h.ServeHTTP(w, req)
	}
}

func BenchmarkHashMiddleware_NoKey(b *testing.B) {
	benchHash(b, "")
}

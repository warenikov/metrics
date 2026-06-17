package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var jsonHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"id":"Alloc","type":"gauge","value":1.5}`)) //nolint:errcheck
})

var plainHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK")) //nolint:errcheck
})

func BenchmarkGzipMiddleware_WithGzip(b *testing.B) {
	b.ReportAllocs()
	h := GzipMiddleware(jsonHandler)
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkGzipMiddleware_NoGzip(b *testing.B) {
	b.ReportAllocs()
	h := GzipMiddleware(plainHandler)
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkLoggerMiddleware(b *testing.B) {
	b.ReportAllocs()
	h := LoggerMiddleware(plainHandler)
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}

func BenchmarkHashMiddleware_NoKey(b *testing.B) {
	b.ReportAllocs()
	h := HashMiddleware("")(plainHandler)
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
}

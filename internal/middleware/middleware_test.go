package middleware

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- GzipMiddleware ---

func TestGzipMiddleware_Response(t *testing.T) {
	tests := []struct {
		name                string
		acceptEncoding      string
		responseContentType string
		expectEncoding      string
		expectCompressed    bool
	}{
		{
			name:                "Без Accept-Encoding — ответ не сжат",
			acceptEncoding:      "",
			responseContentType: "application/json",
			expectEncoding:      "",
			expectCompressed:    false,
		},
		{
			name:                "JSON + gzip — ответ сжат",
			acceptEncoding:      "gzip",
			responseContentType: "application/json",
			expectEncoding:      "gzip",
			expectCompressed:    true,
		},
		{
			name:                "HTML + gzip — ответ сжат",
			acceptEncoding:      "gzip",
			responseContentType: "text/html; charset=utf-8",
			expectEncoding:      "gzip",
			expectCompressed:    true,
		},
	}

	const responseBody = `{"hello":"world"}`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.responseContentType)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(responseBody))
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectEncoding, w.Header().Get("Content-Encoding"))

			if tt.expectCompressed {
				gr, err := gzip.NewReader(w.Body)
				require.NoError(t, err)
				defer gr.Close()
				decoded, err := io.ReadAll(gr)
				require.NoError(t, err)
				assert.Equal(t, responseBody, string(decoded))
			} else {
				assert.Equal(t, responseBody, w.Body.String())
			}
		})
	}
}

func TestGzipMiddleware_DecompressRequest(t *testing.T) {
	const payload = `{"id":"Alloc","type":"gauge","value":100.5}`

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(payload))
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	var received string
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, payload, received)
}

func TestGzipMiddleware_InvalidGzipRequest(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader("this is not gzip"))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- LoggerMiddleware ---

func TestLoggerMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		responseStatus int
		responseBody   string
	}{
		{
			name:           "GET 200",
			method:         http.MethodGet,
			responseStatus: http.StatusOK,
			responseBody:   "ok",
		},
		{
			name:           "POST 404",
			method:         http.MethodPost,
			responseStatus: http.StatusNotFound,
			responseBody:   "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := LoggerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))

			req := httptest.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.responseStatus, w.Code)
			assert.Equal(t, tt.responseBody, w.Body.String())
		})
	}
}

// --- HashMiddleware ---

func testHMAC(body []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestHashMiddleware_NoKey(t *testing.T) {
	handler := HashMiddleware("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
	assert.Empty(t, w.Header().Get("HashSHA256"))
}

func TestHashMiddleware_ValidHash(t *testing.T) {
	const key = "secret"
	body := []byte(`{"id":"Alloc","type":"gauge"}`)

	handler := HashMiddleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("HashSHA256", testHMAC(body, key))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHashMiddleware_InvalidHash(t *testing.T) {
	const key = "secret"
	body := []byte(`{"id":"Alloc","type":"gauge"}`)

	handler := HashMiddleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("HashSHA256", "invalidsignature")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHashMiddleware_NoHashHeader(t *testing.T) {
	const key = "secret"

	handler := HashMiddleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHashMiddleware_ResponseHash(t *testing.T) {
	const key = "secret"
	responseBody := []byte(`{"result":"ok"}`)

	handler := HashMiddleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(responseBody)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, testHMAC(responseBody, key), w.Header().Get("HashSHA256"))
	assert.Equal(t, string(responseBody), w.Body.String())
}

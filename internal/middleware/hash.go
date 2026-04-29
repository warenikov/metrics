package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"metrics/internal/logger"
	"net/http"

	"go.uber.org/zap"
)

type hashResponseWriter struct {
	http.ResponseWriter
	buf    bytes.Buffer
	status int
}

func (w *hashResponseWriter) WriteHeader(code int)        { w.status = code }
func (w *hashResponseWriter) Write(b []byte) (int, error) { return w.buf.Write(b) }

func HashMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			if h := r.Header.Get("HashSHA256"); h != "" {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(body))
				computed := hashBody(body, key)
				logger.Log.Debug("hash check",
					zap.String("received", h),
					zap.String("computed", computed),
				)
				if !hmac.Equal([]byte(h), []byte(computed)) {
					http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
			}

			hrw := &hashResponseWriter{ResponseWriter: w}
			next.ServeHTTP(hrw, r)

			status := hrw.status
			if status == 0 {
				status = http.StatusOK
			}
			w.Header().Set("HashSHA256", hashBody(hrw.buf.Bytes(), key))
			w.WriteHeader(status)
			w.Write(hrw.buf.Bytes()) //nolint:errcheck
		})
	}
}

func hashBody(body []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

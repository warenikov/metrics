package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"go.uber.org/zap"
	"metrics/internal/logger"
	"metrics/pkg/crypto"
)

// maxEncryptedBodySize bounds how much of a request body CryptoMiddleware
// will read before giving up, so an oversized (or infinite, e.g. chunked)
// body can't exhaust server memory. A gzip-compressed metrics batch is
// expected to be at most a few hundred KB even for thousands of metrics.
const maxEncryptedBodySize = 32 << 20 // 32 MiB

// CryptoMiddleware decrypts request bodies encrypted by the agent with the
// public key matching privKey. It runs before GzipMiddleware, since the
// agent encrypts the already gzip-compressed payload.
//
// If privKey is nil, or the request has no body, requests pass through
// unmodified. Body presence is determined by reading, not by ContentLength —
// chunked requests report ContentLength == -1 despite having a body.
func CryptoMiddleware(privKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if privKey == nil {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(io.LimitReader(r.Body, maxEncryptedBodySize+1))
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			if len(body) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			if len(body) > maxEncryptedBodySize {
				http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			}

			plaintext, err := crypto.Decrypt(privKey, body)
			if err != nil {
				logger.Log.Debug("failed to decrypt request body", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plaintext))
			r.ContentLength = int64(len(plaintext))
			next.ServeHTTP(w, r)
		})
	}
}

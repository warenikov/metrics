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

// CryptoMiddleware decrypts request bodies encrypted by the agent with the
// public key matching priv. It runs before GzipMiddleware, since the agent
// encrypts the already gzip-compressed payload.
//
// If priv is nil, or the request has no body, requests pass through
// unmodified.
func CryptoMiddleware(priv *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if priv == nil || r.ContentLength <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			plaintext, err := crypto.Decrypt(priv, body)
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

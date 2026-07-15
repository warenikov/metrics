package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"metrics/pkg/crypto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCryptoMiddleware_NilKey_PassesThrough(t *testing.T) {
	var received []byte
	handler := CryptoMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		received = b
		w.WriteHeader(http.StatusOK)
	}))

	body := []byte("plain body")
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, body, received)
}

func TestCryptoMiddleware_NoBody_PassesThrough(t *testing.T) {
	called := false
	handler := CryptoMiddleware(generateKey(t))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called, "requests without a body must not go through decryption")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCryptoMiddleware_DecryptsBody(t *testing.T) {
	priv := generateKey(t)
	plaintext := []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)
	ciphertext, err := crypto.Encrypt(&priv.PublicKey, plaintext)
	require.NoError(t, err)

	var received []byte
	handler := CryptoMiddleware(priv)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		received = b
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(ciphertext))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, plaintext, received)
}

func TestCryptoMiddleware_BadCiphertext_Returns400(t *testing.T) {
	priv := generateKey(t)
	called := false
	handler := CryptoMiddleware(priv)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader([]byte("not encrypted")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, called)
}

func TestCryptoMiddleware_ChunkedRequest_StillDecrypted(t *testing.T) {
	priv := generateKey(t)
	plaintext := []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)
	ciphertext, err := crypto.Encrypt(&priv.PublicKey, plaintext)
	require.NoError(t, err)

	var received []byte
	handler := CryptoMiddleware(priv)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		received = b
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(ciphertext))
	req.ContentLength = -1 // net/http sets this for Transfer-Encoding: chunked requests
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "a chunked request (ContentLength == -1) must still be decrypted, not skipped")
	assert.Equal(t, plaintext, received)
}

func TestCryptoMiddleware_OversizedBody_Returns413(t *testing.T) {
	priv := generateKey(t)
	called := false
	handler := CryptoMiddleware(priv)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	oversized := bytes.Repeat([]byte{0x01}, maxEncryptedBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(oversized))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.False(t, called)
}

func TestCryptoMiddleware_WrongKey_Returns400(t *testing.T) {
	encryptKey := generateKey(t)
	decryptKey := generateKey(t)
	ciphertext, err := crypto.Encrypt(&encryptKey.PublicKey, []byte("secret"))
	require.NoError(t, err)

	handler := CryptoMiddleware(decryptKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be called on decrypt failure")
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(ciphertext))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func generateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

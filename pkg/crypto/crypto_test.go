package crypto_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"metrics/pkg/crypto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateKeyPair(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func writePEM(t *testing.T, dir, name string, block *pem.Block) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(block), 0o600))
	return path
}

func writePKIXPublicKey(t *testing.T, dir, name string, pub *rsa.PublicKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	return writePEM(t, dir, name, &pem.Block{Type: "PUBLIC KEY", Bytes: der})
}

func writePKCS1PrivateKey(t *testing.T, dir, name string, key *rsa.PrivateKey) string {
	t.Helper()
	der := x509.MarshalPKCS1PrivateKey(key)
	return writePEM(t, dir, name, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
}

func writePKCS8PrivateKey(t *testing.T, dir, name string, key *rsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return writePEM(t, dir, name, &pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := generateKeyPair(t)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{name: "empty", plaintext: []byte{}},
		{name: "short", plaintext: []byte("hello")},
		{name: "json-like", plaintext: []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)},
		{name: "large", plaintext: make([]byte, 100_000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := crypto.Encrypt(&key.PublicKey, tt.plaintext)
			require.NoError(t, err)
			assert.NotEqual(t, tt.plaintext, ciphertext)

			plaintext, err := crypto.Decrypt(key, ciphertext)
			require.NoError(t, err)
			assert.True(t, bytes.Equal(tt.plaintext, plaintext))
		})
	}
}

func TestEncrypt_ProducesDifferentCiphertextEachCall(t *testing.T) {
	key := generateKeyPair(t)
	plaintext := []byte("same input")

	c1, err := crypto.Encrypt(&key.PublicKey, plaintext)
	require.NoError(t, err)
	c2, err := crypto.Encrypt(&key.PublicKey, plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, c1, c2, "random AES key/nonce must make ciphertexts differ")
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	key1 := generateKeyPair(t)
	key2 := generateKeyPair(t)

	ciphertext, err := crypto.Encrypt(&key1.PublicKey, []byte("secret"))
	require.NoError(t, err)

	_, err = crypto.Decrypt(key2, ciphertext)
	assert.Error(t, err)
}

func TestDecrypt_TamperedCiphertextFails(t *testing.T) {
	key := generateKeyPair(t)

	ciphertext, err := crypto.Encrypt(&key.PublicKey, []byte("secret"))
	require.NoError(t, err)
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = crypto.Decrypt(key, ciphertext)
	assert.Error(t, err)
}

func TestDecrypt_TooShortFails(t *testing.T) {
	key := generateKeyPair(t)
	_, err := crypto.Decrypt(key, []byte("short"))
	assert.Error(t, err)
}

func TestLoadPublicKey(t *testing.T) {
	key := generateKeyPair(t)
	dir := t.TempDir()
	path := writePKIXPublicKey(t, dir, "pub.pem", &key.PublicKey)

	pub, err := crypto.LoadPublicKey(path)
	require.NoError(t, err)
	assert.Equal(t, key.PublicKey.N, pub.N)
	assert.Equal(t, key.PublicKey.E, pub.E)
}

func TestLoadPublicKey_Errors(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		_, err := crypto.LoadPublicKey(filepath.Join(dir, "missing.pem"))
		assert.Error(t, err)
	})

	t.Run("not PEM", func(t *testing.T) {
		path := filepath.Join(dir, "notpem.pem")
		require.NoError(t, os.WriteFile(path, []byte("not a pem file"), 0o600))
		_, err := crypto.LoadPublicKey(path)
		assert.Error(t, err)
	})

	t.Run("private key instead of public", func(t *testing.T) {
		key := generateKeyPair(t)
		path := writePKCS1PrivateKey(t, dir, "priv-as-pub.pem", key)
		_, err := crypto.LoadPublicKey(path)
		assert.Error(t, err)
	})
}

func TestLoadPrivateKey_PKCS1(t *testing.T) {
	key := generateKeyPair(t)
	dir := t.TempDir()
	path := writePKCS1PrivateKey(t, dir, "priv.pem", key)

	loaded, err := crypto.LoadPrivateKey(path)
	require.NoError(t, err)
	assert.Equal(t, key.D, loaded.D)
}

func TestLoadPrivateKey_PKCS8(t *testing.T) {
	key := generateKeyPair(t)
	dir := t.TempDir()
	path := writePKCS8PrivateKey(t, dir, "priv8.pem", key)

	loaded, err := crypto.LoadPrivateKey(path)
	require.NoError(t, err)
	assert.Equal(t, key.D, loaded.D)
}

func TestLoadPrivateKey_Errors(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		_, err := crypto.LoadPrivateKey(filepath.Join(dir, "missing.pem"))
		assert.Error(t, err)
	})

	t.Run("not PEM", func(t *testing.T) {
		path := filepath.Join(dir, "notpem.pem")
		require.NoError(t, os.WriteFile(path, []byte("not a pem file"), 0o600))
		_, err := crypto.LoadPrivateKey(path)
		assert.Error(t, err)
	})

	t.Run("public key instead of private", func(t *testing.T) {
		key := generateKeyPair(t)
		path := writePKIXPublicKey(t, dir, "pub-as-priv.pem", &key.PublicKey)
		_, err := crypto.LoadPrivateKey(path)
		assert.Error(t, err)
	})
}

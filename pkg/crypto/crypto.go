// Package crypto provides hybrid RSA+AES encryption for the message bodies
// exchanged between the agent and the server, plus helpers for loading
// PEM-encoded RSA keys from files.
//
// Plain RSA can only encrypt payloads smaller than the key size, so it is
// used here only to wrap a randomly generated AES-256 key; the actual
// payload (which can be an arbitrarily large gzip-compressed JSON batch) is
// encrypted with AES-256-GCM. The wire format produced by [Encrypt] is:
//
//	RSA-OAEP(aesKey) || GCM nonce || GCM ciphertext
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
)

const aesKeySize = 32 // AES-256

// LoadPublicKey reads a PEM-encoded RSA public key (PKIX, e.g. produced by
// `openssl rsa -pubout`) from path.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	block, err := readPEMBlock(path)
	if err != nil {
		return nil, err
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key %s: %w", path, err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%s does not contain an RSA public key", path)
	}
	return rsaPub, nil
}

// LoadPrivateKey reads a PEM-encoded RSA private key (PKCS#1, e.g. produced
// by `openssl genrsa`, or PKCS#8) from path.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	block, err := readPEMBlock(path)
	if err != nil {
		return nil, err
	}
	if key, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes); pkcs1Err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key %s: %w", path, err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%s does not contain an RSA private key", path)
	}
	return rsaKey, nil
}

func readPEMBlock(path string) (*pem.Block, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file %s: %w", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("%s does not contain a valid PEM block", path)
	}
	return block, nil
}

// Encrypt hybrid-encrypts plaintext for pub: a random AES-256 key encrypts
// the payload with AES-GCM, and the AES key itself is encrypted with
// RSA-OAEP so only the holder of the matching private key can recover it.
func Encrypt(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	aesKey := make([]byte, aesKeySize)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, fmt.Errorf("generate AES key: %w", err)
	}

	gcm, err := newGCM(aesKey)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, nonceErr := io.ReadFull(rand.Reader, nonce); nonceErr != nil {
		return nil, fmt.Errorf("generate nonce: %w", nonceErr)
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil) // nonce || ciphertext

	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("encrypt AES key: %w", err)
	}

	return append(encryptedKey, sealed...), nil
}

// Decrypt reverses [Encrypt] using priv.
func Decrypt(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	keySize := priv.Size()
	if len(data) < keySize {
		return nil, fmt.Errorf("ciphertext too short: got %d bytes, need at least %d", len(data), keySize)
	}
	encryptedKey, sealed := data[:keySize], data[keySize:]

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encryptedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt AES key: %w", err)
	}

	gcm, err := newGCM(aesKey)
	if err != nil {
		return nil, err
	}
	if len(sealed) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short for nonce")
	}
	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload: %w", err)
	}
	return plaintext, nil
}

func newGCM(aesKey []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return gcm, nil
}

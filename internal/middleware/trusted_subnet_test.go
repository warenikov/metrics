package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustedSubnetMiddleware_NilSubnet_PassesThrough(t *testing.T) {
	called := false
	handler := TrustedSubnetMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called, "with no trusted subnet configured, requests must not be restricted")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTrustedSubnetMiddleware_IPInSubnet_Allowed(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	called := false
	handler := TrustedSubnetMiddleware(subnet)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.42")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTrustedSubnetMiddleware_IPOutsideSubnet_Forbidden(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	called := false
	handler := TrustedSubnetMiddleware(subnet)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.5")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTrustedSubnetMiddleware_MissingHeader_Forbidden(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	called := false
	handler := TrustedSubnetMiddleware(subnet)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTrustedSubnetMiddleware_UnparseableIP_Forbidden(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	handler := TrustedSubnetMiddleware(subnet)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be called with an unparseable X-Real-IP")
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	req.Header.Set("X-Real-IP", "not-an-ip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

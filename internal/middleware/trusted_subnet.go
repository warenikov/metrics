package middleware

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware rejects requests whose X-Real-IP header is missing,
// unparseable, or outside trusted. If trusted is nil, all requests pass
// through unrestricted.
func TrustedSubnetMiddleware(trusted *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if trusted == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := net.ParseIP(r.Header.Get("X-Real-IP"))
			if ip == nil || !trusted.Contains(ip) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

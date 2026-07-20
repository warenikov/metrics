package middleware

import (
	"net"
	"net/http"

	"metrics/internal/trustedsubnet"
)

// TrustedSubnetMiddleware rejects requests whose X-Real-IP header is missing,
// unparseable, or outside trusted. If trusted is nil, all requests pass
// through unrestricted. The decision itself lives in trustedsubnet.Checker,
// shared with the gRPC interceptor.
func TrustedSubnetMiddleware(trusted *net.IPNet) func(http.Handler) http.Handler {
	checker := trustedsubnet.NewChecker(trusted)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !checker.Allow(r.Header.Get("X-Real-IP")) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Package trustedsubnet holds the transport-independent decision whether a
// client IP belongs to the trusted subnet. The HTTP middleware and the gRPC
// interceptor both delegate here, so the policy lives in one place and the
// transports only translate a rejection into their own error shape.
package trustedsubnet

import "net"

// Checker decides whether a client IP is allowed. The zero Checker has no
// subnet configured and therefore allows everything.
type Checker struct {
	subnet *net.IPNet
}

// NewChecker returns a Checker restricting access to subnet. A nil subnet
// means the check is disabled and every IP is allowed.
func NewChecker(subnet *net.IPNet) Checker {
	return Checker{subnet: subnet}
}

// Allow reports whether rawIP — the textual address taken from the transport
// (an X-Real-IP header or x-real-ip metadata) — belongs to the trusted
// subnet. An empty or unparseable address is never allowed, unless no subnet
// is configured at all.
func (c Checker) Allow(rawIP string) bool {
	if c.subnet == nil {
		return true
	}

	ip := net.ParseIP(rawIP)
	return ip != nil && c.subnet.Contains(ip)
}

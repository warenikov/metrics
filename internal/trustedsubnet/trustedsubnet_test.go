package trustedsubnet_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metrics/internal/trustedsubnet"
)

func TestChecker_NoSubnet_AllowsEverything(t *testing.T) {
	checker := trustedsubnet.NewChecker(nil)

	assert.True(t, checker.Allow("10.0.0.5"))
	assert.True(t, checker.Allow("not-an-ip"))
	assert.True(t, checker.Allow(""))
}

func TestChecker_ZeroValue_AllowsEverything(t *testing.T) {
	var checker trustedsubnet.Checker

	assert.True(t, checker.Allow("10.0.0.5"))
}

func TestChecker_Allow(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	checker := trustedsubnet.NewChecker(subnet)

	tests := []struct {
		name  string
		rawIP string
		want  bool
	}{
		{name: "inside subnet", rawIP: "192.168.1.42", want: true},
		{name: "network address", rawIP: "192.168.1.0", want: true},
		{name: "broadcast address", rawIP: "192.168.1.255", want: true},
		{name: "outside subnet", rawIP: "10.0.0.5", want: false},
		{name: "adjacent subnet", rawIP: "192.168.2.1", want: false},
		{name: "unparseable", rawIP: "not-an-ip", want: false},
		{name: "empty", rawIP: "", want: false},
		{name: "with port", rawIP: "192.168.1.42:8080", want: false},
		{name: "IPv6", rawIP: "::1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, checker.Allow(tt.rawIP))
		})
	}
}

func TestChecker_IPv6Subnet(t *testing.T) {
	_, subnet, err := net.ParseCIDR("fd00::/8")
	require.NoError(t, err)

	checker := trustedsubnet.NewChecker(subnet)

	assert.True(t, checker.Allow("fd00::1"))
	assert.False(t, checker.Allow("fe80::1"))
}

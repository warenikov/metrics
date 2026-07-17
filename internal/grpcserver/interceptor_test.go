package grpcserver

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noopHandler(_ context.Context, req any) (any, error) {
	return req, nil
}

func TestTrustedSubnetInterceptor_Allowed(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "192.168.1.42"))
	interceptor := trustedSubnetInterceptor(subnet)

	resp, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, noopHandler)
	require.NoError(t, err)
	assert.Equal(t, "req", resp)
}

func TestTrustedSubnetInterceptor_OutsideSubnet(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "10.0.0.5"))
	interceptor := trustedSubnetInterceptor(subnet)

	_, err = interceptor(ctx, "req", &grpc.UnaryServerInfo{}, noopHandler)
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestTrustedSubnetInterceptor_MissingMetadata(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	interceptor := trustedSubnetInterceptor(subnet)

	_, err = interceptor(context.Background(), "req", &grpc.UnaryServerInfo{}, noopHandler)
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestTrustedSubnetInterceptor_MissingHeaderKey(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("other-key", "value"))
	interceptor := trustedSubnetInterceptor(subnet)

	_, err = interceptor(ctx, "req", &grpc.UnaryServerInfo{}, noopHandler)
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestTrustedSubnetInterceptor_UnparseableIP(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "not-an-ip"))
	interceptor := trustedSubnetInterceptor(subnet)

	_, err = interceptor(ctx, "req", &grpc.UnaryServerInfo{}, noopHandler)
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

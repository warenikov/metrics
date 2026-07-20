package grpcserver_test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"metrics/internal/grpcserver"
	models "metrics/internal/model"
	pb "metrics/internal/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubUpdater struct {
	batches [][]models.Metrics
}

func (s *stubUpdater) UpdateBatch(_ context.Context, metrics []models.Metrics) error {
	s.batches = append(s.batches, metrics)
	return nil
}

// startTestServer starts a grpcserver.Server on a free loopback port,
// serves it in the background, and returns a client connection dialed
// against it. Verifies New()'s own wiring (interceptor registration,
// service registration), not just the interceptor/handler in isolation.
func startTestServer(t *testing.T, updater grpcserver.MetricsUpdater, trustedSubnet *net.IPNet) pb.MetricsClient {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close()) // free the port for Start() to rebind

	srv := grpcserver.New(addr, updater, trustedSubnet)
	go func() { _ = srv.Start() }()
	t.Cleanup(srv.Stop)

	// Give Start() a moment to bind before dialing.
	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if dialErr != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 2*time.Second, 20*time.Millisecond)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return pb.NewMetricsClient(conn)
}

func TestGRPCServer_UpdateMetrics_NoTrustedSubnet(t *testing.T) {
	updater := &stubUpdater{}
	client := startTestServer(t, updater, nil)

	resp, err := client.UpdateMetrics(context.Background(), pb.UpdateMetricsRequest_builder{
		Metrics: []*pb.Metric{pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42}.Build()},
	}.Build())
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, updater.batches, 1)
	assert.Equal(t, "Alloc", updater.batches[0][0].ID)
}

func TestGRPCServer_TrustedSubnet_RejectsOutsideIP(t *testing.T) {
	updater := &stubUpdater{}
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)
	client := startTestServer(t, updater, subnet)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-real-ip", "10.0.0.1")
	_, err = client.UpdateMetrics(ctx, pb.UpdateMetricsRequest_builder{
		Metrics: []*pb.Metric{pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42}.Build()},
	}.Build())

	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Empty(t, updater.batches, "handler must not be reached when the interceptor rejects the request")
}

func TestGRPCServer_TrustedSubnet_AllowsInsideIP(t *testing.T) {
	updater := &stubUpdater{}
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)
	client := startTestServer(t, updater, subnet)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-real-ip", "192.168.1.7")
	resp, err := client.UpdateMetrics(ctx, pb.UpdateMetricsRequest_builder{
		Metrics: []*pb.Metric{pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42}.Build()},
	}.Build())

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, updater.batches, 1)
}

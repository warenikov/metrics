// Package grpcserver implements the gRPC transport for the metrics server:
// the Metrics service defined in api/metrics.proto, a trusted-subnet
// interceptor, and graceful shutdown.
package grpcserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"

	models "metrics/internal/model"
	pb "metrics/internal/proto"
)

// gracefulStopTimeout bounds how long Stop waits for in-flight RPCs to
// finish before forcibly closing the server.
const gracefulStopTimeout = 5 * time.Second

// MetricsUpdater is the interface for writing a batch of metrics to the store.
type MetricsUpdater interface {
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
}

// Server wraps a gRPC server exposing the Metrics service.
type Server struct {
	grpcServer *grpc.Server
	addr       string
}

// New creates a Server listening on addr. If trustedSubnet is nil, requests
// are accepted regardless of their x-real-ip metadata.
func New(addr string, updater MetricsUpdater, trustedSubnet *net.IPNet) *Server {
	var opts []grpc.ServerOption
	if trustedSubnet != nil {
		opts = append(opts, grpc.ChainUnaryInterceptor(trustedSubnetInterceptor(trustedSubnet)))
	}

	s := grpc.NewServer(opts...)
	pb.RegisterMetricsServer(s, &metricsServer{updater: updater})

	return &Server{grpcServer: s, addr: addr}
}

// Start begins listening for incoming gRPC requests. It blocks until the
// server is stopped.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.addr, err)
	}
	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the server, waiting for in-flight RPCs to finish.
// If they don't finish within gracefulStopTimeout, it forcibly stops instead.
func (s *Server) Stop() {
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(gracefulStopTimeout):
		s.grpcServer.Stop()
	}
}

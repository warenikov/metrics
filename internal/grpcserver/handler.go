package grpcserver

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metrics/internal/logger"
	pb "metrics/internal/proto"
	"metrics/internal/protoconv"
	"metrics/internal/service"
)

type metricsServer struct {
	pb.UnimplementedMetricsServer
	updater MetricsUpdater
}

// UpdateMetrics stores the metrics carried by req. It is safe to call with
// zero, one, or many metrics — the same method handles both single-metric
// and batch updates.
func (s *metricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	protoMetrics := req.GetMetrics()
	if len(protoMetrics) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	if err := s.updater.UpdateBatch(ctx, protoconv.FromProto(protoMetrics)); err != nil {
		return nil, mapUpdateError(err)
	}
	return &pb.UpdateMetricsResponse{}, nil
}

// mapUpdateError translates a repository/service error into a gRPC status,
// mirroring the HTTP handlers' writeError mapping. Internal error details
// are logged server-side, never sent to the client.
func mapUpdateError(err error) error {
	switch {
	case errors.Is(err, service.ErrTypeMetric), errors.Is(err, service.ErrInvalidValue):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, service.ErrMetricNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		logger.Log.Error("failed to update metrics via gRPC", zap.Error(err))
		return status.Error(codes.Internal, "internal error")
	}
}

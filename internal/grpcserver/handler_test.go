package grpcserver

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	models "metrics/internal/model"
	pb "metrics/internal/proto"
	"metrics/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUpdater struct {
	err      error
	received []models.Metrics
}

func (f *fakeUpdater) UpdateBatch(_ context.Context, metrics []models.Metrics) error {
	f.received = metrics
	return f.err
}

func TestMetricsServer_UpdateMetrics_Success(t *testing.T) {
	updater := &fakeUpdater{}
	srv := &metricsServer{updater: updater}

	req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1.5}.Build(),
		pb.Metric_builder{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 3}.Build(),
	}}.Build()

	resp, err := srv.UpdateMetrics(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	require.Len(t, updater.received, 2)
	assert.Equal(t, "Alloc", updater.received[0].ID)
	assert.Equal(t, models.Gauge, updater.received[0].MType)
	assert.Equal(t, "PollCount", updater.received[1].ID)
	assert.Equal(t, models.Counter, updater.received[1].MType)
}

func TestMetricsServer_UpdateMetrics_EmptyBatch_NoOp(t *testing.T) {
	updater := &fakeUpdater{}
	srv := &metricsServer{updater: updater}

	resp, err := srv.UpdateMetrics(context.Background(), pb.UpdateMetricsRequest_builder{}.Build())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Nil(t, updater.received, "updater must not be called for an empty batch")
}

func TestMetricsServer_UpdateMetrics_ErrorMapping(t *testing.T) {
	tests := []struct {
		updaterErr error
		name       string
		wantCode   codes.Code
	}{
		{name: "invalid value", updaterErr: service.ErrInvalidValue, wantCode: codes.InvalidArgument},
		{name: "invalid type", updaterErr: service.ErrTypeMetric, wantCode: codes.InvalidArgument},
		{name: "not found", updaterErr: service.ErrMetricNotFound, wantCode: codes.NotFound},
		{name: "unknown error", updaterErr: errors.New("boom"), wantCode: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater := &fakeUpdater{err: tt.updaterErr}
			srv := &metricsServer{updater: updater}

			_, err := srv.UpdateMetrics(context.Background(), pb.UpdateMetricsRequest_builder{
				Metrics: []*pb.Metric{pb.Metric_builder{Id: "x", Type: pb.Metric_GAUGE}.Build()},
			}.Build())

			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestMetricsServer_UpdateMetrics_InternalErrorNotLeaked(t *testing.T) {
	updater := &fakeUpdater{err: errors.New("sensitive db connection string leaked here")}
	srv := &metricsServer{updater: updater}

	_, err := srv.UpdateMetrics(context.Background(), pb.UpdateMetricsRequest_builder{
		Metrics: []*pb.Metric{pb.Metric_builder{Id: "x", Type: pb.Metric_GAUGE}.Build()},
	}.Build())

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "sensitive", "internal error details must not reach the client")
}

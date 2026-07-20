package protoconv_test

import (
	"testing"

	models "metrics/internal/model"
	pb "metrics/internal/proto"
	"metrics/internal/protoconv"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToProto(t *testing.T) {
	value := 1.5
	delta := int64(42)

	metrics := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &value},
		{ID: "PollCount", MType: models.Counter, Delta: &delta},
		{ID: "NilValue", MType: models.Gauge, Value: nil},
	}

	out := protoconv.ToProto(metrics)
	require.Len(t, out, 3)

	assert.Equal(t, "Alloc", out[0].GetId())
	assert.Equal(t, pb.Metric_GAUGE, out[0].GetType())
	assert.InDelta(t, 1.5, out[0].GetValue(), 0.0001)

	assert.Equal(t, "PollCount", out[1].GetId())
	assert.Equal(t, pb.Metric_COUNTER, out[1].GetType())
	assert.Equal(t, int64(42), out[1].GetDelta())

	assert.Equal(t, "NilValue", out[2].GetId())
	assert.Equal(t, float64(0), out[2].GetValue(), "nil Value must convert to the zero value, not panic")
}

func TestFromProto(t *testing.T) {
	in := []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 2.5}.Build(),
		pb.Metric_builder{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 7}.Build(),
	}

	out := protoconv.FromProto(in)
	require.Len(t, out, 2)

	assert.Equal(t, "Alloc", out[0].ID)
	assert.Equal(t, models.Gauge, out[0].MType)
	require.NotNil(t, out[0].Value)
	assert.InDelta(t, 2.5, *out[0].Value, 0.0001)
	assert.Nil(t, out[0].Delta)

	assert.Equal(t, "PollCount", out[1].ID)
	assert.Equal(t, models.Counter, out[1].MType)
	require.NotNil(t, out[1].Delta)
	assert.Equal(t, int64(7), *out[1].Delta)
	assert.Nil(t, out[1].Value)
}

func TestRoundTrip(t *testing.T) {
	value := 3.14
	delta := int64(-5)
	original := []models.Metrics{
		{ID: "g", MType: models.Gauge, Value: &value},
		{ID: "c", MType: models.Counter, Delta: &delta},
	}

	roundTripped := protoconv.FromProto(protoconv.ToProto(original))
	require.Len(t, roundTripped, 2)

	assert.Equal(t, original[0].ID, roundTripped[0].ID)
	assert.Equal(t, original[0].MType, roundTripped[0].MType)
	assert.InDelta(t, *original[0].Value, *roundTripped[0].Value, 0.0001)

	assert.Equal(t, original[1].ID, roundTripped[1].ID)
	assert.Equal(t, original[1].MType, roundTripped[1].MType)
	assert.Equal(t, *original[1].Delta, *roundTripped[1].Delta)
}

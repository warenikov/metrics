// Package protoconv converts between the internal models.Metrics
// representation and the generated protobuf Metric type used by the gRPC
// transport, so the conversion logic lives in one place shared by the
// agent (encoding) and the server (decoding).
package protoconv

import (
	models "metrics/internal/model"
	pb "metrics/internal/proto"
)

// ToProto converts a batch of internal metrics to their protobuf representation.
func ToProto(metrics []models.Metrics) []*pb.Metric {
	out := make([]*pb.Metric, len(metrics))
	for i, m := range metrics {
		b := pb.Metric_builder{Id: m.ID}
		switch m.MType {
		case models.Gauge:
			b.Type = pb.Metric_GAUGE
			if m.Value != nil {
				b.Value = *m.Value
			}
		case models.Counter:
			b.Type = pb.Metric_COUNTER
			if m.Delta != nil {
				b.Delta = *m.Delta
			}
		}
		out[i] = b.Build()
	}
	return out
}

// FromProto converts a batch of protobuf metrics back into the internal
// representation. Delta/Value are always populated (never nil) since the
// wire format carries them as plain scalars, not optional fields.
func FromProto(metrics []*pb.Metric) []models.Metrics {
	out := make([]models.Metrics, len(metrics))
	for i, pm := range metrics {
		m := models.Metrics{ID: pm.GetId()}
		switch pm.GetType() {
		case pb.Metric_GAUGE:
			m.MType = models.Gauge
			value := pm.GetValue()
			m.Value = &value
		case pb.Metric_COUNTER:
			m.MType = models.Counter
			delta := pm.GetDelta()
			m.Delta = &delta
		}
		out[i] = m
	}
	return out
}

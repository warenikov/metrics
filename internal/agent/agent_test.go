package agent

import (
	"runtime"
	"testing"
)

// BenchmarkCollectMetricsReflection замеряет текущую реализацию с reflect
func BenchmarkCollectMetricsReflection(b *testing.B) {
	m := &MetricaAgent{
		ms:       &runtime.MemStats{},
		gauges:   &GaugeMertics{},
		counters: &CounterMertics{},
	}
	runtime.ReadMemStats(m.ms)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.collectMerticsReflection()
	}
}

// BenchmarkCollectMetricsManual замеряет прямое присваивание
func BenchmarkCollectMetricsManual(b *testing.B) {
	m := &MetricaAgent{
		ms:       &runtime.MemStats{},
		gauges:   &GaugeMertics{},
		counters: &CounterMertics{},
	}
	runtime.ReadMemStats(m.ms)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.collectMerticsManual()
	}
}

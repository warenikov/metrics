package agent

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	models "metrics/internal/model"

	"github.com/stretchr/testify/assert"
)

func TestCollectBatchIncludesGopsutilMetrics(t *testing.T) {
	m := &MetricaAgent{
		ms: &runtime.MemStats{},
		gauges: &GaugeMertics{
			TotalMemory: 8 * 1024 * 1024 * 1024,
			FreeMemory:  4 * 1024 * 1024 * 1024,
		},
		counters:       &CounterMertics{},
		cpuUtilization: []float64{10.5, 20.3},
	}

	batch := m.collectBatch()

	names := make(map[string]bool, len(batch))
	for _, metric := range batch {
		names[metric.ID] = true
	}

	assert.True(t, names["TotalMemory"], "TotalMemory missing from batch")
	assert.True(t, names["FreeMemory"], "FreeMemory missing from batch")
	assert.True(t, names["CPUutilization1"], "CPUutilization1 missing from batch")
	assert.True(t, names["CPUutilization2"], "CPUutilization2 missing from batch")
}

func TestWorkerPoolRespectsRateLimit(t *testing.T) {
	const rateLimit = 2

	var current, maxSeen atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := current.Add(1)
		for {
			prev := maxSeen.Load()
			if c <= prev || maxSeen.CompareAndSwap(prev, c) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		current.Add(-1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := &MetricaAgent{
		ms:         &runtime.MemStats{},
		gauges:     &GaugeMertics{},
		counters:   &CounterMertics{},
		serverAddr: srv.URL,
		rateLimit:  rateLimit,
	}

	jobs := make(chan []models.Metrics, rateLimit)

	var wg sync.WaitGroup
	for i := 0; i < rateLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.worker(jobs)
		}()
	}

	v := 1.0
	batch := []models.Metrics{{ID: "test", MType: models.Gauge, Value: &v}}
	for i := 0; i < 10; i++ {
		jobs <- batch
	}
	close(jobs)
	wg.Wait()

	assert.LessOrEqual(t, maxSeen.Load(), int32(rateLimit),
		"concurrent requests exceeded rateLimit")
}

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

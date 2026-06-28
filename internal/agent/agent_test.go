package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"metrics/internal/config"
	models "metrics/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectBatchIncludesGopsutilMetrics(t *testing.T) {
	m := &MetricaAgent{
		ms: &runtime.MemStats{},
		gauges: &models.GaugeMertics{
			TotalMemory: 8 * 1024 * 1024 * 1024,
			FreeMemory:  4 * 1024 * 1024 * 1024,
		},
		counters:       &models.CounterMertics{},
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
		gauges:     &models.GaugeMertics{},
		counters:   &models.CounterMertics{},
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

func TestComputeHMAC(t *testing.T) {
	got := computeHMAC([]byte("hello"), "secret")
	assert.NotEmpty(t, got)
	assert.Equal(t, got, computeHMAC([]byte("hello"), "secret"))
	assert.NotEqual(t, computeHMAC([]byte("a"), "k"), computeHMAC([]byte("b"), "k"))
}

func TestIsRetryableNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	resp, netErr := http.Get(url) //nolint:noctx
	if resp != nil {
		resp.Body.Close()
	}
	require.Error(t, netErr)
	assert.True(t, isRetryableNetworkError(netErr))
	assert.False(t, isRetryableNetworkError(errors.New("plain error")))
}

func TestResolveKey(t *testing.T) {
	assert.Equal(t, "", resolveKey(""))
	assert.Equal(t, "mykey", resolveKey("mykey"))
	assert.Equal(t, "", resolveKey("/nonexistent/path/to/key"))

	f, err := os.CreateTemp(t.TempDir(), "key")
	require.NoError(t, err)
	f.Close()
	assert.Equal(t, f.Name(), resolveKey(f.Name()))
}

func TestNewMetricaAgent(t *testing.T) {
	cfg := &config.Config{
		ServerAddr:     "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		RateLimit:      3,
	}
	a := NewMetricaAgent(cfg)
	require.NotNil(t, a)
	assert.Equal(t, "http://localhost:8080", a.serverAddr)
	assert.Equal(t, 2*time.Second, a.pollInterval)
	assert.Equal(t, 10*time.Second, a.sendInterval)
	assert.Equal(t, 3, a.rateLimit)
}

func TestPoll(t *testing.T) {
	m := &MetricaAgent{
		ms:       &runtime.MemStats{},
		gauges:   &models.GaugeMertics{},
		counters: &models.CounterMertics{},
	}
	require.NoError(t, m.Poll())
	assert.Greater(t, m.gauges.Alloc, 0.0)
}

func TestCollectMerticsReflection(t *testing.T) {
	m := &MetricaAgent{
		ms:       &runtime.MemStats{},
		gauges:   &models.GaugeMertics{},
		counters: &models.CounterMertics{},
	}
	runtime.ReadMemStats(m.ms)
	m.collectMerticsReflection()
	assert.Greater(t, m.gauges.Alloc, 0.0)
	assert.Equal(t, int64(1), m.counters.PollCount)
}

func TestCollectMerticsManual(t *testing.T) {
	m := &MetricaAgent{
		ms:       &runtime.MemStats{},
		gauges:   &models.GaugeMertics{},
		counters: &models.CounterMertics{},
	}
	runtime.ReadMemStats(m.ms)
	m.collectMerticsManual()
	assert.Greater(t, m.gauges.Alloc, 0.0)
	assert.Equal(t, int64(29), m.counters.PollCount)
}

func TestSend_PostsToServer(t *testing.T) {
	var reqCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := &MetricaAgent{
		ms:         &runtime.MemStats{},
		gauges:     &models.GaugeMertics{},
		counters:   &models.CounterMertics{},
		serverAddr: srv.URL,
	}
	runtime.ReadMemStats(m.ms)
	m.collectMerticsManual()
	m.Send()

	assert.Greater(t, reqCount.Load(), int32(0))
}

func TestSendBatch_PostsToServer(t *testing.T) {
	var reqCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := &MetricaAgent{
		ms:         &runtime.MemStats{},
		gauges:     &models.GaugeMertics{},
		counters:   &models.CounterMertics{},
		serverAddr: srv.URL,
	}
	runtime.ReadMemStats(m.ms)
	m.collectMerticsManual()
	m.SendBatch()

	assert.Equal(t, int32(1), reqCount.Load())
}

func TestPostRequest_RetriesOnNetworkError(t *testing.T) {
	origDelays := agentRetryDelays
	agentRetryDelays = []time.Duration{1 * time.Millisecond, 1 * time.Millisecond, 1 * time.Millisecond}
	defer func() { agentRetryDelays = origDelays }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	m := &MetricaAgent{serverAddr: url}
	m.postRequest([]byte(`{}`), "test")
}

func TestPollRuntime_CancelContext(t *testing.T) {
	m := &MetricaAgent{
		ms:           &runtime.MemStats{},
		gauges:       &models.GaugeMertics{},
		counters:     &models.CounterMertics{},
		pollInterval: 1 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		m.pollRuntime(ctx)
		close(done)
	}()
	time.Sleep(5 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Error("pollRuntime did not return after context cancellation")
	}
}

func TestPollGopsutil_CancelContext(t *testing.T) {
	m := &MetricaAgent{
		ms:           &runtime.MemStats{},
		gauges:       &models.GaugeMertics{},
		counters:     &models.CounterMertics{},
		pollInterval: 1 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		m.pollGopsutil(ctx)
		close(done)
	}()
	time.Sleep(5 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Error("pollGopsutil did not return after context cancellation")
	}
}

// BenchmarkCollectMetricsReflection замеряет текущую реализацию с reflect
func BenchmarkCollectMetricsReflection(b *testing.B) {
	m := &MetricaAgent{
		ms:       &runtime.MemStats{},
		gauges:   &models.GaugeMertics{},
		counters: &models.CounterMertics{},
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
		gauges:   &models.GaugeMertics{},
		counters: &models.CounterMertics{},
	}
	runtime.ReadMemStats(m.ms)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.collectMerticsManual()
	}
}

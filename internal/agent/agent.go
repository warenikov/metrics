package agent

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"metrics/internal/config"
	"metrics/internal/logger"
	models "metrics/internal/model"
	pb "metrics/internal/proto"
	"metrics/internal/protoconv"
	"metrics/pkg/compress"
	"metrics/pkg/crypto"
	"net"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	ErrSendMetrica = errors.New("failed to send metrica")
)

var agentRetryDelays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// shutdownGrace bounds how long workers may keep sending after the agent
// context is cancelled. It lets them drain the batches already queued instead
// of dropping them, while still capping shutdown: without it a worker could
// hang for the request timeout plus the whole retry backoff.
const shutdownGrace = 5 * time.Second

// sleepCtx waits for d and reports whether the wait completed. It returns
// false as soon as ctx is done, so retry backoff never outlives cancellation.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func computeHMAC(body []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func isRetryableNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

// isRetryableGRPCError reports whether err represents a transient gRPC
// failure worth retrying, as opposed to a terminal one (e.g. PermissionDenied,
// InvalidArgument) that would fail identically on every retry.
func isRetryableGRPCError(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
		return true
	default:
		return false
	}
}

// localIP returns the outbound IP address of this host: the source address
// the OS would use to reach an external host. Dialing UDP performs no
// handshake and sends no packets, so this is safe to call even without
// network connectivity to the target.
func localIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("determine local IP: %w", err)
	}
	defer func() { _ = conn.Close() }()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", fmt.Errorf("determine local IP: unexpected local address type %T", conn.LocalAddr())
	}
	return addr.IP.String(), nil
}

type MetricaAgent struct {
	ms             *runtime.MemStats
	gauges         *models.GaugeMertics
	counters       *models.CounterMertics
	pubKey         *rsa.PublicKey
	grpcClient     pb.MetricsClient
	serverAddr     string
	key            string
	hostIP         string
	cpuUtilization []float64
	pollInterval   time.Duration
	sendInterval   time.Duration
	rateLimit      int
	mu             sync.RWMutex
}

func resolveKey(key string) string {
	if strings.HasPrefix(key, "/") {
		if _, err := os.Stat(key); err != nil {
			return ""
		}
	}
	return key
}

func NewMetricaAgent(cfg *config.Config, pubKey *rsa.PublicKey, grpcClient pb.MetricsClient) *MetricaAgent {
	srv := fmt.Sprintf("http://%s", cfg.ServerAddr)
	hostIP, err := localIP()
	if err != nil {
		logger.Log.Error("failed to determine local IP for X-Real-IP header", zap.Error(err))
	}
	return &MetricaAgent{
		ms:           &runtime.MemStats{},
		gauges:       &models.GaugeMertics{},
		counters:     &models.CounterMertics{},
		pollInterval: time.Duration(cfg.PollInterval) * time.Second,
		sendInterval: time.Duration(cfg.ReportInterval) * time.Second,
		serverAddr:   srv,
		key:          resolveKey(cfg.Key),
		hostIP:       hostIP,
		pubKey:       pubKey,
		grpcClient:   grpcClient,
		rateLimit:    cfg.RateLimit,
	}
}

func (m *MetricaAgent) Run(ctx context.Context) {
	rateLimit := m.rateLimit
	if rateLimit <= 0 {
		rateLimit = 1
	}

	jobs := make(chan []models.Metrics, rateLimit)

	// Workers outlive ctx by shutdownGrace: once the agent is asked to stop,
	// the sender queues nothing more, but whatever is already in flight or in
	// the queue still gets a bounded window to reach the server.
	sendCtx, cancelSend := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelSend()
	stopGrace := context.AfterFunc(ctx, func() {
		time.AfterFunc(shutdownGrace, cancelSend)
	})
	defer stopGrace()

	var wg sync.WaitGroup
	for i := 0; i < rateLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.worker(sendCtx, jobs)
		}()
	}

	go m.pollRuntime(ctx)
	go m.pollGopsutil(ctx)
	m.sender(ctx, jobs)

	close(jobs)
	wg.Wait()
}

func (m *MetricaAgent) pollRuntime(ctx context.Context) {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			runtime.ReadMemStats(m.ms)
			m.collectMerticsManual()
			m.mu.Unlock()
		}
	}
}

func (m *MetricaAgent) pollGopsutil(ctx context.Context) {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			vm, memErr := mem.VirtualMemory()
			cpuPercents, cpuErr := cpu.Percent(0, true)

			m.mu.Lock()
			if memErr == nil {
				m.gauges.TotalMemory = float64(vm.Total)
				m.gauges.FreeMemory = float64(vm.Free)
			} else {
				logger.Log.Error("gopsutil mem error", zap.Error(memErr))
			}
			if cpuErr == nil {
				m.cpuUtilization = cpuPercents
			} else {
				logger.Log.Error("gopsutil cpu error", zap.Error(cpuErr))
			}
			m.mu.Unlock()
		}
	}
}

func (m *MetricaAgent) sender(ctx context.Context, jobs chan<- []models.Metrics) {
	ticker := time.NewTicker(m.sendInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			batch := m.collectBatch()
			m.counters.PollCount = 0
			m.mu.Unlock()

			select {
			case jobs <- batch:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (m *MetricaAgent) worker(ctx context.Context, jobs <-chan []models.Metrics) {
	for batch := range jobs {
		m.sendBatch(ctx, batch)
	}
}

// sendBatch sends batch via gRPC if a client is configured, otherwise falls
// back to the HTTP JSON transport.
func (m *MetricaAgent) sendBatch(ctx context.Context, batch []models.Metrics) {
	if m.grpcClient != nil {
		m.sendBatchGRPC(ctx, batch)
		return
	}
	data, err := json.Marshal(batch)
	if err != nil {
		logger.Log.Error("failed to marshal batch", zap.Error(err))
		return
	}
	m.postBatchRequest(ctx, data)
}

func (m *MetricaAgent) Poll() error {
	runtime.ReadMemStats(m.ms)
	logger.Log.Debug("Poll metrics")
	m.collectMerticsManual()

	return nil
}

func (m *MetricaAgent) collectMerticsReflection() {
	src := reflect.ValueOf(m.ms).Elem()
	dst := reflect.ValueOf(m.gauges).Elem()
	for i := 0; i < dst.NumField(); i++ {
		fieldName := dst.Type().Field(i).Name
		fSrc := src.FieldByName(fieldName)
		if fSrc.IsValid() {
			switch fSrc.Kind() {
			case reflect.Uint64, reflect.Uint32:
				dst.Field(i).SetFloat(float64(fSrc.Uint()))
			case reflect.Float64:
				dst.Field(i).SetFloat(fSrc.Float())
			default:
			}
		}
	}

	m.gauges.RandomValue = rand.Float64()
	m.counters.PollCount += 1
}

func (m *MetricaAgent) collectMerticsManual() {
	m.gauges.Alloc = float64(m.ms.Alloc)
	m.gauges.BuckHashSys = float64(m.ms.BuckHashSys)
	m.gauges.Frees = float64(m.ms.Frees)
	m.gauges.GCCPUFraction = m.ms.GCCPUFraction
	m.gauges.MSpanSys = float64(m.ms.MSpanSys)
	m.gauges.GCSys = float64(m.ms.GCSys)
	m.gauges.HeapAlloc = float64(m.ms.HeapAlloc)
	m.gauges.HeapIdle = float64(m.ms.HeapIdle)
	m.gauges.HeapInuse = float64(m.ms.HeapInuse)
	m.gauges.HeapObjects = float64(m.ms.HeapObjects)
	m.gauges.HeapReleased = float64(m.ms.HeapReleased)
	m.gauges.HeapSys = float64(m.ms.HeapSys)
	m.gauges.LastGC = float64(m.ms.LastGC)
	m.gauges.Lookups = float64(m.ms.Lookups)
	m.gauges.MCacheInuse = float64(m.ms.MCacheInuse)
	m.gauges.MCacheSys = float64(m.ms.MCacheSys)
	m.gauges.MSpanInuse = float64(m.ms.MSpanInuse)
	m.gauges.Mallocs = float64(m.ms.Mallocs)
	m.gauges.NextGC = float64(m.ms.NextGC)
	m.gauges.NumForcedGC = float64(m.ms.NumForcedGC)
	m.gauges.NumGC = float64(m.ms.NumGC)
	m.gauges.OtherSys = float64(m.ms.OtherSys)
	m.gauges.PauseTotalNs = float64(m.ms.PauseTotalNs)
	m.gauges.StackInuse = float64(m.ms.StackInuse)
	m.gauges.StackSys = float64(m.ms.StackSys)
	m.gauges.Sys = float64(m.ms.Sys)
	m.gauges.TotalAlloc = float64(m.ms.TotalAlloc)

	m.gauges.RandomValue = rand.Float64()
	m.counters.PollCount += 29 //29 потому что 28 метрик + сам каунтер
}

func (m *MetricaAgent) Send(ctx context.Context) {
	g := m.gauges
	m.sendGauge(ctx, "Alloc", g.Alloc)
	m.sendGauge(ctx, "BuckHashSys", g.BuckHashSys)
	m.sendGauge(ctx, "Frees", g.Frees)
	m.sendGauge(ctx, "GCCPUFraction", g.GCCPUFraction)
	m.sendGauge(ctx, "GCSys", g.GCSys)
	m.sendGauge(ctx, "HeapAlloc", g.HeapAlloc)
	m.sendGauge(ctx, "HeapIdle", g.HeapIdle)
	m.sendGauge(ctx, "HeapInuse", g.HeapInuse)
	m.sendGauge(ctx, "HeapObjects", g.HeapObjects)
	m.sendGauge(ctx, "HeapReleased", g.HeapReleased)
	m.sendGauge(ctx, "HeapSys", g.HeapSys)
	m.sendGauge(ctx, "LastGC", g.LastGC)
	m.sendGauge(ctx, "Lookups", g.Lookups)
	m.sendGauge(ctx, "MCacheInuse", g.MCacheInuse)
	m.sendGauge(ctx, "MCacheSys", g.MCacheSys)
	m.sendGauge(ctx, "MSpanInuse", g.MSpanInuse)
	m.sendGauge(ctx, "MSpanSys", g.MSpanSys)
	m.sendGauge(ctx, "Mallocs", g.Mallocs)
	m.sendGauge(ctx, "NextGC", g.NextGC)
	m.sendGauge(ctx, "NumForcedGC", g.NumForcedGC)
	m.sendGauge(ctx, "NumGC", g.NumGC)
	m.sendGauge(ctx, "OtherSys", g.OtherSys)
	m.sendGauge(ctx, "PauseTotalNs", g.PauseTotalNs)
	m.sendGauge(ctx, "StackInuse", g.StackInuse)
	m.sendGauge(ctx, "StackSys", g.StackSys)
	m.sendGauge(ctx, "Sys", g.Sys)
	m.sendGauge(ctx, "TotalAlloc", g.TotalAlloc)
	m.sendGauge(ctx, "RandomValue", g.RandomValue)

	// Отправляем Counter метрики
	m.sendCounter(ctx, "PollCount", m.counters.PollCount)

	logger.Log.Debug("Send metrics")
}

func (m *MetricaAgent) collectBatch() []models.Metrics {
	g := m.gauges
	gaugeVal := func(name string, v float64) models.Metrics {
		val := v
		return models.Metrics{ID: name, MType: models.Gauge, Value: &val}
	}
	batch := []models.Metrics{
		gaugeVal("Alloc", g.Alloc),
		gaugeVal("BuckHashSys", g.BuckHashSys),
		gaugeVal("Frees", g.Frees),
		gaugeVal("GCCPUFraction", g.GCCPUFraction),
		gaugeVal("GCSys", g.GCSys),
		gaugeVal("HeapAlloc", g.HeapAlloc),
		gaugeVal("HeapIdle", g.HeapIdle),
		gaugeVal("HeapInuse", g.HeapInuse),
		gaugeVal("HeapObjects", g.HeapObjects),
		gaugeVal("HeapReleased", g.HeapReleased),
		gaugeVal("HeapSys", g.HeapSys),
		gaugeVal("LastGC", g.LastGC),
		gaugeVal("Lookups", g.Lookups),
		gaugeVal("MCacheInuse", g.MCacheInuse),
		gaugeVal("MCacheSys", g.MCacheSys),
		gaugeVal("MSpanInuse", g.MSpanInuse),
		gaugeVal("MSpanSys", g.MSpanSys),
		gaugeVal("Mallocs", g.Mallocs),
		gaugeVal("NextGC", g.NextGC),
		gaugeVal("NumForcedGC", g.NumForcedGC),
		gaugeVal("NumGC", g.NumGC),
		gaugeVal("OtherSys", g.OtherSys),
		gaugeVal("PauseTotalNs", g.PauseTotalNs),
		gaugeVal("StackInuse", g.StackInuse),
		gaugeVal("StackSys", g.StackSys),
		gaugeVal("Sys", g.Sys),
		gaugeVal("TotalAlloc", g.TotalAlloc),
		gaugeVal("RandomValue", g.RandomValue),
	}
	batch = append(batch,
		gaugeVal("TotalMemory", g.TotalMemory),
		gaugeVal("FreeMemory", g.FreeMemory),
	)
	for i, v := range m.cpuUtilization {
		batch = append(batch, gaugeVal(fmt.Sprintf("CPUutilization%d", i+1), v))
	}
	delta := m.counters.PollCount
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &delta})
	return batch
}

func (m *MetricaAgent) SendBatch(ctx context.Context) {
	batch := m.collectBatch()
	m.sendBatch(ctx, batch)
	logger.Log.Debug("SendBatch metrics")
}

// sendBatchGRPC sends batch to the server over gRPC, retrying transient
// failures with the same backoff schedule as the HTTP transport.
func (m *MetricaAgent) sendBatchGRPC(ctx context.Context, batch []models.Metrics) {
	req := &pb.UpdateMetricsRequest{Metrics: protoconv.ToProto(batch)}

	doRequest := func() error {
		reqCtx := ctx
		if m.hostIP != "" {
			reqCtx = metadata.AppendToOutgoingContext(reqCtx, "x-real-ip", m.hostIP)
		}
		reqCtx, cancel := context.WithTimeout(reqCtx, 30*time.Second)
		defer cancel()

		_, err := m.grpcClient.UpdateMetrics(reqCtx, req)
		return err
	}

	lastErr := doRequest()
	for _, d := range agentRetryDelays {
		if lastErr == nil || !isRetryableGRPCError(lastErr) {
			break
		}
		if !sleepCtx(ctx, d) {
			break
		}
		lastErr = doRequest()
	}
	if lastErr != nil {
		logger.Log.Error("failed to send batch via gRPC", zap.Error(lastErr))
	}
}

// prepareBody gzip-compresses data and, if a public key is configured,
// encrypts the compressed payload so only the server holding the matching
// private key can read it.
func (m *MetricaAgent) prepareBody(data []byte) (io.Reader, error) {
	compressedData, err := compress.Compress(data)
	if err != nil {
		return nil, err
	}
	if m.pubKey == nil {
		return compressedData, nil
	}
	encrypted, err := crypto.Encrypt(m.pubKey, compressedData.Bytes())
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(encrypted), nil
}

func (m *MetricaAgent) postBatchRequest(ctx context.Context, data []byte) {
	var lastErr error
	client := &http.Client{Timeout: 30 * time.Second}

	doRequest := func() error {
		body, err := m.prepareBody(data)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.serverAddr+"/updates/", body)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")
		if m.hostIP != "" {
			req.Header.Set("X-Real-IP", m.hostIP)
		}
		if m.key != "" {
			req.Header.Set("HashSHA256", computeHMAC(data, m.key))
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		reader := io.Reader(resp.Body)
		if resp.Header.Get("Content-Encoding") == "gzip" {
			gz, gzErr := compress.NewReader(resp.Body)
			if gzErr != nil {
				return gzErr
			}
			defer func() { _ = gz.Close() }()
			reader = gz
		}
		_, err = io.ReadAll(reader)
		return err
	}

	lastErr = doRequest()
	for _, d := range agentRetryDelays {
		if lastErr == nil || !isRetryableNetworkError(lastErr) {
			break
		}
		if !sleepCtx(ctx, d) {
			break
		}
		lastErr = doRequest()
	}
	if lastErr != nil {
		logger.Log.Error("failed to send batch", zap.Error(lastErr))
	}
}

// sendGauge Вспомогательный метод для отправки Gauge
func (m *MetricaAgent) sendGauge(ctx context.Context, name string, value float64) {
	data, err := json.Marshal(&models.Metrics{ID: name, Value: &value, MType: models.Gauge})
	if err != nil {
		logger.Log.Error("failed to marshal gauge", zap.Error(err))
		return
	}
	m.postRequest(ctx, data, name)
}

// sendCounter Вспомогательный метод для отправки Counter
func (m *MetricaAgent) sendCounter(ctx context.Context, name string, value int64) {
	data, err := json.Marshal(&models.Metrics{ID: name, Delta: &value, MType: models.Counter})
	if err != nil {
		logger.Log.Error("failed to marshal counter", zap.Error(err))
		return
	}
	m.postRequest(ctx, data, name)
}

func (m *MetricaAgent) postRequest(ctx context.Context, data []byte, name string) {
	var lastErr error
	client := &http.Client{Timeout: 30 * time.Second}

	doRequest := func() error {
		body, err := m.prepareBody(data)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.serverAddr+"/update/", body)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")
		if m.hostIP != "" {
			req.Header.Set("X-Real-IP", m.hostIP)
		}
		if m.key != "" {
			req.Header.Set("HashSHA256", computeHMAC(data, m.key))
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		reader := io.Reader(resp.Body)
		if resp.Header.Get("Content-Encoding") == "gzip" {
			gz, gzErr := compress.NewReader(resp.Body)
			if gzErr != nil {
				return gzErr
			}
			defer func() { _ = gz.Close() }()
			reader = gz
		}
		_, err = io.ReadAll(reader)
		return err
	}

	lastErr = doRequest()
	for _, d := range agentRetryDelays {
		if lastErr == nil || !isRetryableNetworkError(lastErr) {
			break
		}
		if !sleepCtx(ctx, d) {
			break
		}
		lastErr = doRequest()
	}
	if lastErr != nil {
		logger.Log.Error("failed to send metric",
			zap.String("metrica name", name),
			zap.Error(lastErr),
		)
	}
}

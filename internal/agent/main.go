package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"metrics/internal/config"
	"metrics/internal/logger"
	models "metrics/internal/model"
	"metrics/pkg/compress"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"time"

	"go.uber.org/zap"
)

var (
	ErrSendMetrica = errors.New("failed to send metrica")
)

var agentRetryDelays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

func isRetryableNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

type GaugeMertics struct {
	Alloc         float64 `json:"alloc"`
	BuckHashSys   float64 `json:"buck_hash_sys"`
	Frees         float64 `json:"frees"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`
	GCSys         float64 `json:"gc_sys"`
	HeapAlloc     float64 `json:"heap_alloc"`
	HeapIdle      float64 `json:"heap_idle"`
	HeapInuse     float64 `json:"heap_inuse"`
	HeapObjects   float64 `json:"heap_objects"`
	HeapReleased  float64 `json:"heap_released"`
	HeapSys       float64 `json:"heap_sys"`
	LastGC        float64 `json:"last_gc"`
	Lookups       float64 `json:"lookups"`
	MCacheInuse   float64 `json:"mcache_inuse"`
	MCacheSys     float64 `json:"mcache_sys"`
	MSpanInuse    float64 `json:"mspan_inuse"`
	MSpanSys      float64 `json:"mspan_sys"`
	Mallocs       float64 `json:"mallocs"`
	NextGC        float64 `json:"next_gc"`
	NumForcedGC   float64 `json:"num_forced_gc"`
	NumGC         float64 `json:"num_gc"`
	OtherSys      float64 `json:"other_sys"`
	PauseTotalNs  float64 `json:"pause_total_ns"`
	StackInuse    float64 `json:"stack_inuse"`
	StackSys      float64 `json:"stack_sys"`
	Sys           float64 `json:"sys"`
	TotalAlloc    float64 `json:"total_alloc"`
	RandomValue   float64 `json:"random_value"`
}

type CounterMertics struct {
	PollCount int64 `json:"poll_count"`
}

type MetricaAgent struct {
	ms           *runtime.MemStats
	gauges       *GaugeMertics
	counters     *CounterMertics
	pollInterval time.Duration
	sendInterval time.Duration
	serverAddr   string
}

func NewMetricaAgent(cfg *config.Config) *MetricaAgent {
	srv := fmt.Sprintf("http://%s", cfg.ServerAddr)
	return &MetricaAgent{
		ms:           &runtime.MemStats{},
		gauges:       &GaugeMertics{},
		counters:     &CounterMertics{},
		pollInterval: time.Duration(cfg.PollInterval) * time.Second,
		sendInterval: time.Duration(cfg.ReportInterval) * time.Second,
		serverAddr:   srv,
	}
}

func (m *MetricaAgent) Run() {
	var timePassed time.Duration
	for {
		_ = m.Poll()
		timePassed += m.pollInterval
		if timePassed >= m.sendInterval {
			m.SendBatch()
			m.counters.PollCount = 0 //обнуляем каунтер после отправки
			timePassed = 0
		}
		time.Sleep(m.pollInterval)
	}
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

func (m *MetricaAgent) Send() {
	g := m.gauges
	m.sendGauge("Alloc", g.Alloc)
	m.sendGauge("BuckHashSys", g.BuckHashSys)
	m.sendGauge("Frees", g.Frees)
	m.sendGauge("GCCPUFraction", g.GCCPUFraction)
	m.sendGauge("GCSys", g.GCSys)
	m.sendGauge("HeapAlloc", g.HeapAlloc)
	m.sendGauge("HeapIdle", g.HeapIdle)
	m.sendGauge("HeapInuse", g.HeapInuse)
	m.sendGauge("HeapObjects", g.HeapObjects)
	m.sendGauge("HeapReleased", g.HeapReleased)
	m.sendGauge("HeapSys", g.HeapSys)
	m.sendGauge("LastGC", g.LastGC)
	m.sendGauge("Lookups", g.Lookups)
	m.sendGauge("MCacheInuse", g.MCacheInuse)
	m.sendGauge("MCacheSys", g.MCacheSys)
	m.sendGauge("MSpanInuse", g.MSpanInuse)
	m.sendGauge("MSpanSys", g.MSpanSys)
	m.sendGauge("Mallocs", g.Mallocs)
	m.sendGauge("NextGC", g.NextGC)
	m.sendGauge("NumForcedGC", g.NumForcedGC)
	m.sendGauge("NumGC", g.NumGC)
	m.sendGauge("OtherSys", g.OtherSys)
	m.sendGauge("PauseTotalNs", g.PauseTotalNs)
	m.sendGauge("StackInuse", g.StackInuse)
	m.sendGauge("StackSys", g.StackSys)
	m.sendGauge("Sys", g.Sys)
	m.sendGauge("TotalAlloc", g.TotalAlloc)
	m.sendGauge("RandomValue", g.RandomValue)

	// Отправляем Counter метрики
	m.sendCounter("PollCount", m.counters.PollCount)

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
	delta := m.counters.PollCount
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &delta})
	return batch
}

func (m *MetricaAgent) SendBatch() {
	batch := m.collectBatch()
	data, err := json.Marshal(batch)
	if err != nil {
		logger.Log.Error("failed to marshal batch", zap.Error(err))
		return
	}
	m.postBatchRequest(data)
	logger.Log.Debug("SendBatch metrics")
}

func (m *MetricaAgent) postBatchRequest(data []byte) {
	var lastErr error
	client := &http.Client{}

	doRequest := func() error {
		compressedData, err := compress.Compress(data)
		if err != nil {
			return err
		}
		req, err := http.NewRequest("POST", m.serverAddr+"/updates/", compressedData)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		reader := io.Reader(resp.Body)
		if resp.Header.Get("Content-Encoding") == "gzip" {
			gz, err := compress.NewReader(resp.Body)
			if err != nil {
				return err
			}
			defer gz.Close()
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
		time.Sleep(d)
		lastErr = doRequest()
	}
	if lastErr != nil {
		logger.Log.Error("failed to send batch", zap.Error(lastErr))
	}
}

// sendGauge Вспомогательный метод для отправки Gauge
func (m *MetricaAgent) sendGauge(name string, value float64) {
	data, err := json.Marshal(&models.Metrics{ID: name, Value: &value, MType: models.Gauge})
	if err != nil {
		logger.Log.Error("failed to marshal gauge", zap.Error(err))
		return
	}
	m.postRequest(data, name)
}

// sendCounter Вспомогательный метод для отправки Counter
func (m *MetricaAgent) sendCounter(name string, value int64) {
	data, err := json.Marshal(&models.Metrics{ID: name, Delta: &value, MType: models.Counter})
	if err != nil {
		logger.Log.Error("failed to marshal counter", zap.Error(err))
		return
	}
	m.postRequest(data, name)
}

func (m *MetricaAgent) postRequest(data []byte, name string) {
	var lastErr error
	client := &http.Client{}

	doRequest := func() error {
		compressedData, err := compress.Compress(data)
		if err != nil {
			return err
		}
		req, err := http.NewRequest("POST", m.serverAddr+"/update/", compressedData)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		reader := io.Reader(resp.Body)
		if resp.Header.Get("Content-Encoding") == "gzip" {
			gz, err := compress.NewReader(resp.Body)
			if err != nil {
				return err
			}
			defer gz.Close()
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
		time.Sleep(d)
		lastErr = doRequest()
	}
	if lastErr != nil {
		logger.Log.Error("failed to send metric",
			zap.String("metrica name", name),
			zap.Error(lastErr),
		)
	}
}

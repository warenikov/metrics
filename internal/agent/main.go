package agent

import (
	"fmt"
	"math/rand/v2"
	"metrics/internal/config"
	models "metrics/internal/model"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"
)

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
	Malloc        float64 `json:"malloc"`
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
	rm         sync.RWMutex
	ms         *runtime.MemStats
	gauges     *GaugeMertics
	counters   *CounterMertics
	pollTicker *time.Ticker
	sendTicker *time.Ticker
	serverAddr string
}

func NewMetricaAgent(cfg *config.Config) *MetricaAgent {
	srv := fmt.Sprintf("http://%s:%s", cfg.ServerAddr, cfg.ServerPort)
	return &MetricaAgent{
		ms:         &runtime.MemStats{},
		gauges:     &GaugeMertics{},
		counters:   &CounterMertics{},
		pollTicker: time.NewTicker(time.Duration(cfg.PollInterval) * time.Second),
		sendTicker: time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second),
		serverAddr: srv,
	}
}

func (m *MetricaAgent) Run() {
	for {
		select {
		case <-m.pollTicker.C:
			m.Poll()
			//TODO: тут логирование
			fmt.Println("Метрики собраны")
		case <-m.sendTicker.C:
			m.Send()
			//TODO: тут логирование
			fmt.Println("Метрики отправлены")
		}
	}
}

func (m *MetricaAgent) Poll() error {
	runtime.ReadMemStats(m.ms)

	m.collectMertics()

	return nil
}

func (m *MetricaAgent) collectMertics() {
	m.rm.Lock()
	defer m.rm.Unlock()

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
			}
		}
	}

	m.gauges.RandomValue = rand.Float64()
	m.counters.PollCount += 1
}

func (m *MetricaAgent) Send() {
	gauge := reflect.ValueOf(m.gauges).Elem()

	for i := 0; i < gauge.NumField(); i++ {
		fieldName := gauge.Type().Field(i).Name
		fieldValue := gauge.Field(i)
		url := fmt.Sprintf("/update/gauge/%s/%s", fieldName, strconv.FormatFloat(fieldValue.Float(), 'f', -1, 64))
		m.sender(m.serverAddr, url)
	}

	counters := reflect.ValueOf(m.counters).Elem()

	for i := 0; i < counters.NumField(); i++ {
		fieldName := counters.Type().Field(i).Name
		fieldValue := counters.Field(i)
		url := fmt.Sprintf("/update/counter/%s/%d", fieldName, fieldValue.Int())
		m.sender(m.serverAddr, url)
	}
}

func (m *MetricaAgent) sender(addr, url string) bool {
	r, e := http.Post(fmt.Sprintf("%s%s", addr, url), models.ContentTypeText, nil)
	if e != nil {
		fmt.Printf("Ошибка при отправке метрик %s , ошибка %v\n", url, e)
		return false
	}

	if r.StatusCode != http.StatusOK {
		fmt.Printf("Ошибка при отправке метрик %s , код ответа севрера %d\n", url, r.StatusCode)
		return false
	} else {
		fmt.Printf("Метрика отправлена %s\n", url)
		return true
	}

	return true
}

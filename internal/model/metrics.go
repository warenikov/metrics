// Package models defines the core data types for the metrics server.
package models

import (
	"strconv"
)

// Metric type identifiers used in API requests and storage.
const (
	// Counter is an additive metric type whose value increases monotonically.
	Counter = "counter"
	// Gauge is a metric type that represents an instantaneous measurement.
	Gauge = "gauge"
)

// GaugeMertics holds all runtime and system gauge metric values collected by the agent.
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
	TotalMemory   float64 `json:"total_memory"`
	FreeMemory    float64 `json:"free_memory"`
}

// CounterMertics holds counter metric values collected by the agent.
type CounterMertics struct {
	PollCount int64 `json:"poll_count"`
}

// Metrics is the unified representation of a single metric used in the HTTP API.
//
// Delta and Value are pointers to distinguish a zero value from an absent value
// during JSON serialisation (omitempty). Use MType to determine which field is
// meaningful: [Counter] metrics carry Delta, [Gauge] metrics carry Value.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

// ValueString returns the metric value as a human-readable string.
// For gauge metrics it formats the float64; for counter metrics the int64.
// Returns an empty string for unknown metric types and "0" when the value pointer is nil.
func (m Metrics) ValueString() string {
	switch m.MType {
	case "gauge":
		if m.Value == nil {
			return "0"
		}
		return strconv.FormatFloat(*m.Value, 'f', -1, 64)
	case "counter":
		if m.Delta == nil {
			return "0"
		}
		return strconv.FormatInt(*m.Delta, 10)
	default:
		return ""
	}
}

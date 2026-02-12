// Package models defines data structures for metrics representation.
package models

const (
	// Counter represents a counter metric type (monotonically increasing integer).
	Counter = "counter"
	// Gauge represents a gauge metric type (any floating-point value).
	Gauge = "gauge"
)

// Metrics represents a single metric with its metadata and value.
//
// Structure details:
//   - ID: Unique metric identifier (e.g., "cpu_usage", "memory_alloc")
//   - MType: Metric type - either "gauge" or "counter"
//   - Delta: Used for counter metrics (pointer to distinguish 0 from unset)
//   - Value: Used for gauge metrics (pointer to distinguish 0 from unset)
//   - Hash: Optional HMAC-SHA256 signature for request authentication
//
// Example gauge metric:
//
//	{"id": "temperature", "type": "gauge", "value": 23.5}
//
// Example counter metric:
//
//	{"id": "requests", "type": "counter", "delta": 1000}
type Metrics struct {
	ID    string   `json:"id"`              // Unique metric identifier
	MType string   `json:"type"`            // Metric type: "gauge" or "counter"
	Delta *int64   `json:"delta,omitempty"` // Counter value (nil for gauges)
	Value *float64 `json:"value,omitempty"` // Gauge value (nil for counters)
	Hash  string   `json:"hash,omitempty"`  // HMAC-SHA256 signature
}

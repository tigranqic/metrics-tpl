// Package audit provides asynchronous audit event publishing to multiple observers.
// It supports multiple audit backends: file-based logging and HTTP endpoints.
package audit

// Event represents an audit log entry for metric updates.
// Each time metrics are updated, an Event is created and sent to all registered observers.
//
// Fields:
//   - Timestamp: Unix timestamp of when the update occurred
//   - Metrics: List of metric IDs that were updated
//   - IPAddress: IP address of the client that requested the update
//
// Example event JSON:
//
//	{"ts": 1707400000, "metrics": ["cpu_usage", "memory_usage"], "ip_address": "192.168.1.100"}
type Event struct {
	Timestamp int64    `json:"ts"`         // Unix timestamp
	Metrics   []string `json:"metrics"`    // Metric IDs that were updated
	IPAddress string   `json:"ip_address"` // Client IP address
}

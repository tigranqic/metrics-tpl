package audit_test

import (
	"fmt"
	"time"

	"github.com/tigranqic/metrics-tpl/internal/audit"
)

// Example_createAuditEvent demonstrates creating an audit event.
// Audit events record when and which metrics were updated.
func Example_createAuditEvent() {
	event := audit.Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"cpu_usage", "memory_usage"},
		IPAddress: "192.168.1.100",
	}

	fmt.Printf("Event for metrics: %v\n", event.Metrics)
	fmt.Printf("From IP: %s\n", event.IPAddress)
	// Output:
	// Event for metrics: [cpu_usage memory_usage]
	// From IP: 192.168.1.100
}

// Example_multipleMetricsEvent demonstrates an audit event with multiple metrics.
func Example_multipleMetricsEvent() {
	event := audit.Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"gauge_1", "gauge_2", "counter_1", "counter_2"},
		IPAddress: "10.0.0.5",
	}

	fmt.Printf("Updated %d metrics from %s\n", len(event.Metrics), event.IPAddress)
	// Output:
	// Updated 4 metrics from 10.0.0.5
}

// Example_auditEventWithBatchUpdate demonstrates an audit event from batch operations.
// When multiple metrics are updated in one request, they all appear in one audit event.
func Example_auditEventWithBatchUpdate() {
	batchMetrics := []string{"temperature", "humidity", "pressure"}

	event := audit.Event{
		Timestamp: time.Now().Unix(),
		Metrics:   batchMetrics,
		IPAddress: "172.16.0.1",
	}

	for _, metric := range event.Metrics {
		fmt.Printf("Metric updated: %s (IP: %s)\n", metric, event.IPAddress)
	}
	// Output:
	// Metric updated: temperature (IP: 172.16.0.1)
	// Metric updated: humidity (IP: 172.16.0.1)
	// Metric updated: pressure (IP: 172.16.0.1)
}

// Example_auditEventIPAddress demonstrates tracking the source IP address.
// This helps identify which client made metric updates.
func Example_auditEventIPAddress() {
	clientIP := "192.0.2.123"
	event := audit.Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"agent_uptime"},
		IPAddress: clientIP,
	}

	fmt.Printf("Metric update from client: %s\n", event.IPAddress)
	// Output:
	// Metric update from client: 192.0.2.123
}

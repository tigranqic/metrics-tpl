package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http/httptest"

	"github.com/tigranqic/metrics-tpl/internal/handler"
	models "github.com/tigranqic/metrics-tpl/internal/model"
	"github.com/tigranqic/metrics-tpl/internal/repository"
	"go.uber.org/zap"
)

// Example_updateGaugeMetric demonstrates updating a gauge metric using URL parameters.
// This example shows the simplest way to update a metric by using the REST API.
func Example_updateGaugeMetric() {
	// Create in-memory storage and handler
	store := repository.NewMemStorage("", 0)
	logger := zap.NewNop()
	h := handler.NewHandler(store, nil, logger, "", nil)

	// Create a test HTTP request to update a gauge metric
	req := httptest.NewRequest("POST", "/update/gauge/temperature/23.5", nil)
	w := httptest.NewRecorder()

	// Handle the request
	h.Router().ServeHTTP(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	// Output:
	// Status: 200
}

// Example_updateCounterMetric demonstrates updating a counter metric using URL parameters.
// Counters accumulate - each update adds to the existing value.
func Example_updateCounterMetric() {
	store := repository.NewMemStorage("", 0)
	logger := zap.NewNop()
	h := handler.NewHandler(store, nil, logger, "", nil)

	// Update counter metric three times
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/update/counter/requests/10", nil)
		w := httptest.NewRecorder()
		h.Router().ServeHTTP(w, req)
	}

	// Retrieve the counter value
	req := httptest.NewRequest("GET", "/value/counter/requests", nil)
	w := httptest.NewRecorder()
	h.Router().ServeHTTP(w, req)

	fmt.Printf("Counter value: %s\n", w.Body.String())
	// Output:
	// Counter value: 30
}

// Example_updateJSONMetric demonstrates updating a metric using JSON request body.
// This approach is useful when you need to send additional data like hash signatures.
func Example_updateJSONMetric() {
	store := repository.NewMemStorage("", 0)
	logger := zap.NewNop()
	h := handler.NewHandler(store, nil, logger, "", nil)

	// Create a metric update request in JSON format
	metric := models.Metrics{
		ID:    "cpu_usage",
		MType: "gauge",
		Value: ptrFloat64(75.5),
	}

	body, _ := json.Marshal(metric)
	req := httptest.NewRequest("POST", "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Router().ServeHTTP(w, req)

	// Parse the response
	var response models.Metrics
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		fmt.Printf("Failed to decode response: %v\n", err)
		return
	}

	fmt.Printf("Updated metric: %s (type: %s, value: %.1f)\n",
		response.ID, response.MType, *response.Value)
	// Output:
	// Updated metric: cpu_usage (type: gauge, value: 75.5)
}

// Example_batchUpdateMetrics demonstrates updating multiple metrics in a single request.
// Batch updates are atomic - all metrics are updated successfully or none are.
func Example_batchUpdateMetrics() {
	store := repository.NewMemStorage("", 0)
	logger := zap.NewNop()
	h := handler.NewHandler(store, nil, logger, "", nil)

	// Create multiple metrics
	metrics := []models.Metrics{
		{
			ID:    "memory_alloc",
			MType: "gauge",
			Value: ptrFloat64(512.5),
		},
		{
			ID:    "goroutines",
			MType: "gauge",
			Value: ptrFloat64(25.0),
		},
		{
			ID:    "requests_processed",
			MType: "counter",
			Delta: ptrInt64(1000),
		},
	}

	body, _ := json.Marshal(metrics)
	req := httptest.NewRequest("POST", "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Router().ServeHTTP(w, req)

	fmt.Printf("Batch update status: %d\n", w.Code)
	// Output:
	// Batch update status: 200
}

// Example_getMetricValue demonstrates retrieving a metric value.
// Supports both gauge and counter metrics.
func Example_getMetricValue() {
	store := repository.NewMemStorage("", 0)
	logger := zap.NewNop()
	h := handler.NewHandler(store, nil, logger, "", nil)

	// First, set a gauge metric
	updateReq := httptest.NewRequest("POST", "/update/gauge/cpu/45.5", nil)
	updateW := httptest.NewRecorder()
	h.Router().ServeHTTP(updateW, updateReq)

	// Then retrieve it
	getReq := httptest.NewRequest("GET", "/value/gauge/cpu", nil)
	w := httptest.NewRecorder()
	h.Router().ServeHTTP(w, getReq)

	fmt.Printf("Retrieved metric value: %s\n", w.Body.String())
	// Output:
	// Retrieved metric value: 45.5
}

// Example_listAllMetrics demonstrates retrieving all metrics in HTML format.
// This endpoint is useful for monitoring dashboards and debugging.
func Example_listAllMetrics() {
	store := repository.NewMemStorage("", 0)
	logger := zap.NewNop()
	h := handler.NewHandler(store, nil, logger, "", nil)

	// Add some metrics
	if err := store.Update("gauge", "temp", "22.5"); err != nil {
		log.Fatal(err)
	}
	if err := store.Update("counter", "requests", "100"); err != nil {
		log.Fatal(err)
	}

	// List all metrics
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.Router().ServeHTTP(w, req)

	// Check response contains metric names
	if bytes.Contains(w.Body.Bytes(), []byte("temp")) {
		fmt.Println("Metrics listing includes 'temp' metric")
	}
	// Output:
	// Metrics listing includes 'temp' metric
}

// Example_healthCheck demonstrates checking service health and database connectivity.
func Example_healthCheck() {
	store := repository.NewMemStorage("", 0)
	logger := zap.NewNop()
	h := handler.NewHandler(store, nil, logger, "", nil)

	// Check health
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.Router().ServeHTTP(w, req)

	fmt.Printf("Health check status: %d\n", w.Code)
	// Output:
	// Health check status: 200
}

// Example_errorHandling demonstrates error cases when sending invalid data.
// The API validates metric types and values.
func Example_errorHandling() {
	store := repository.NewMemStorage("", 0)
	logger := zap.NewNop()
	h := handler.NewHandler(store, nil, logger, "", nil)

	// Try to update with invalid gauge value (non-numeric)
	req := httptest.NewRequest("POST", "/update/gauge/temp/invalid", nil)
	w := httptest.NewRecorder()
	h.Router().ServeHTTP(w, req)

	fmt.Printf("Invalid gauge update status: %d\n", w.Code)

	// Try to update with invalid metric type
	req = httptest.NewRequest("POST", "/update/unknown/metric/123", nil)
	w = httptest.NewRecorder()
	h.Router().ServeHTTP(w, req)

	fmt.Printf("Invalid metric type status: %d\n", w.Code)
	// Output:
	// Invalid gauge update status: 400
	// Invalid metric type status: 400
}

// Helper functions for creating pointers to primitive types
func ptrFloat64(v float64) *float64 {
	return &v
}

func ptrInt64(v int64) *int64 {
	return &v
}

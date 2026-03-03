package repository_test

import (
	"fmt"
	"log"
	"sort"

	models "github.com/tigranqic/metrics-tpl/internal/model"
	"github.com/tigranqic/metrics-tpl/internal/repository"
)

// Example_updateGaugeMetric demonstrates how to update a gauge metric in storage.
// Gauge metrics can be set to any value (they don't accumulate).
func Example_updateGaugeMetric() {
	// Create in-memory storage (no persistence due to empty path and zero interval)
	store := repository.NewMemStorage("", 0)

	// Update a gauge metric
	err := store.Update("gauge", "cpu_usage", "45.5")
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve the metric we just set
	value, err := store.GetGauge("cpu_usage")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("CPU usage: %.1f\n", value)
	// Output:
	// CPU usage: 45.5
}

// Example_updateCounterMetric demonstrates how to update a counter metric.
// Counter metrics accumulate - each update adds to the existing value.
func Example_updateCounterMetric() {
	store := repository.NewMemStorage("", 0)

	// First update
	if err := store.Update("counter", "requests", "100"); err != nil {
		log.Fatal(err)
	}

	// Second update - adds to existing value
	if err := store.Update("counter", "requests", "50"); err != nil {
		log.Fatal(err)
	}

	// Retrieve the counter
	value, err := store.GetCounter("requests")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total requests: %d\n", value)
	// Output:
	// Total requests: 150
}

// Example_batchUpdate demonstrates updating multiple metrics atomically in a single operation.
// All metrics in a batch are updated together (all succeed or all fail).
func Example_batchUpdate() {
	store := repository.NewMemStorage("", 0)

	// Create multiple metrics
	batch := []models.Metrics{
		{
			ID:    "memory_usage",
			MType: "gauge",
			Value: ptrFloat64(512.75),
		},
		{
			ID:    "goroutines",
			MType: "gauge",
			Value: ptrFloat64(25.0),
		},
		{
			ID:    "cache_hits",
			MType: "counter",
			Delta: ptrInt64(1000),
		},
	}

	// Update all metrics in a single atomic operation
	err := store.UpdateBatch(batch)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve the metrics
	memory, _ := store.GetGauge("memory_usage")
	goroutines, _ := store.GetGauge("goroutines")
	hits, _ := store.GetCounter("cache_hits")

	fmt.Printf("Memory: %.2f MB, Goroutines: %.0f, Cache hits: %d\n",
		memory, goroutines, hits)
	// Output:
	// Memory: 512.75 MB, Goroutines: 25, Cache hits: 1000
}

// Example_getAllMetrics demonstrates retrieving all stored metrics.
// Returns a map of metric IDs to Metrics objects.
func Example_getAllMetrics() {
	store := repository.NewMemStorage("", 0)

	// Add several metrics
	if err := store.Update("gauge", "cpu", "75.5"); err != nil {
		log.Fatal(err)
	}
	if err := store.Update("gauge", "memory", "512.0"); err != nil {
		log.Fatal(err)
	}
	if err := store.Update("counter", "requests", "1000"); err != nil {
		log.Fatal(err)
	}

	// Get all metrics
	metrics, err := store.GetAll()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total metrics stored: %d\n", len(metrics))

	// Sort keys for consistent output
	keys := make([]string, 0, len(metrics))
	for k := range metrics {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Iterate through metrics in sorted order
	for _, id := range keys {
		metric := metrics[id]
		switch metric.MType {
		case "gauge":
			fmt.Printf("%s (gauge): %.1f\n", id, *metric.Value)
		case "counter":
			fmt.Printf("%s (counter): %d\n", id, *metric.Delta)
		}
	}
	// Output:
	// Total metrics stored: 3
	// cpu (gauge): 75.5
	// memory (gauge): 512.0
	// requests (counter): 1000
}

// Example_errorHandling demonstrates handling errors when metrics don't exist or have wrong type.
func Example_errorHandling() {
	store := repository.NewMemStorage("", 0)

	// Try to get a gauge metric that doesn't exist
	_, err := store.GetGauge("nonexistent")
	if err != nil {
		fmt.Printf("Error getting nonexistent gauge: %v\n", err)
	}

	// Add a gauge metric
	if err := store.Update("gauge", "temp", "22.5"); err != nil {
		log.Fatal(err)
	}

	// Try to get it as a counter (wrong type)
	_, err = store.GetCounter("temp")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Correct retrieval works
	value, _ := store.GetGauge("temp")
	fmt.Printf("Correct retrieval: %.1f\n", value)
	// Output:
	// Error getting nonexistent gauge: gauge not found
	// Error: counter not found
	// Correct retrieval: 22.5
}

// Example_filePersistence demonstrates automatic metric persistence to file.
// Metrics are saved to a JSON file at regular intervals.
func Example_filePersistence() {
	// Create storage with file persistence (save every 5 seconds)
	store := repository.NewMemStorage("/tmp/metrics.json", 5)

	// Add metrics
	if err := store.Update("gauge", "temperature", "23.5"); err != nil {
		log.Fatal(err)
	}
	if err := store.Update("counter", "events", "42"); err != nil {
		log.Fatal(err)
	}
	// Metrics are periodically saved to /tmp/metrics.json
	// You can load them from the file on next application start

	// Manual save to file
	err := store.SaveToFile("/tmp/metrics.json")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Metrics persisted to file")
	// Output:
	// Metrics persisted to file
}

// Example_concurrentUpdates demonstrates that storage handles concurrent updates safely.
// The mutex ensures thread-safe access to metrics.
func Example_concurrentUpdates() {
	store := repository.NewMemStorage("", 0)

	// In a real application, you would use goroutines:
	// go store.Update("counter", "requests", "1")
	// go store.Update("counter", "requests", "1")
	// go store.Update("gauge", "cpu", "50")

	// For this example, we'll do sequential updates
	for i := 0; i < 5; i++ {
		if err := store.Update("counter", "requests", "10"); err != nil {
			log.Fatal(err)
		}
	}

	count, _ := store.GetCounter("requests")
	fmt.Printf("Final counter value (50 from 5x10): %d\n", count)
	// Output:
	// Final counter value (50 from 5x10): 50
}

// Helper functions for creating pointers
func ptrFloat64(v float64) *float64 {
	return &v
}

func ptrInt64(v int64) *int64 {
	return &v
}

package repository

import (
	"os"
	"testing"

	"github.com/tigranqic/metrics-tpl/pkg/logger"
)

// TestMemStorageShutdown verifies that MemStorage.Shutdown saves all metrics to file
func TestMemStorageShutdown(t *testing.T) {
	logger.Init("debug", "text")

	tmpFile := "/tmp/test_metrics_shutdown.json"
	defer func() {
		if err := os.Remove(tmpFile); err != nil && !os.IsNotExist(err) {
			t.Fatalf("failed to remove tmp file: %v", err)
		}
	}()
	// Create MemStorage with file persistence
	store := NewMemStorage(tmpFile, 0) // syncWrite = true (storeInterval = 0)

	// Add some metrics
	if err := store.Update("gauge", "temp", "42.5"); err != nil {
		t.Fatalf("failed to update metric: %v", err)
	}

	if err := store.Update("counter", "requests", "100"); err != nil {
		t.Fatalf("failed to update counter: %v", err)
	}

	// Shutdown should save to file
	if err := store.Shutdown(); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	// Verify file was created and contains data
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("metrics file was not created during shutdown")
	}

	// Load a new store from the file and verify data
	store2 := NewMemStorage(tmpFile, 0)

	// Load from file
	if err := store2.LoadFromFile(tmpFile); err != nil {
		t.Fatalf("failed to load from file: %v", err)
	}

	// Verify metrics were persisted
	if v, err := store2.GetGauge("temp"); err != nil || v != 42.5 {
		t.Fatalf("gauge metric not properly persisted: got %v, want 42.5", v)
	}

	if v, err := store2.GetCounter("requests"); err != nil || v != 100 {
		t.Fatalf("counter metric not properly persisted: got %v, want 100", v)
	}
}

// TestMemStorageShutdownNoFile verifies Shutdown is no-op when filePath is empty
func TestMemStorageShutdownNoFile(t *testing.T) {
	logger.Init("debug", "text")

	// Create MemStorage without file persistence
	store := NewMemStorage("", 0)

	// Add some metrics
	if err := store.Update("gauge", "temp", "42.5"); err != nil {
		t.Fatalf("failed to update metric: %v", err)
	}

	// Shutdown should succeed even without file path
	if err := store.Shutdown(); err != nil {
		t.Fatalf("shutdown with no file path failed: %v", err)
	}
}

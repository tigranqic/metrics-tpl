// Package repository provides storage abstraction layer for metrics.
package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	models "github.com/tigranqic/metrics-tpl/internal/model"
)

// Storage defines the interface for metric storage implementations.
// Implementations must support both gauge and counter metrics,
// single and batch operations, and provide atomicity for batch updates.
type Storage interface {
	// Update updates or creates a single metric.
	// For gauge metrics: replaces the existing value.
	// For counter metrics: adds the delta to existing value (or creates new if not exists).
	// Parameters:
	//   - metricType: "gauge" or "counter"
	//   - name: metric identifier
	//   - value: string representation of the value to update
	// Returns error if metric type is invalid or value cannot be parsed.
	Update(metricType, name, value string) error

	// UpdateBatch updates multiple metrics in a single operation.
	// For gauge metrics: replaces values.
	// For counter metrics: adds deltas to existing values.
	// All metrics in the batch are updated together (all succeed or all fail).
	// Parameters:
	//   - batch: slice of metrics to update
	// Returns error if batch contains invalid data or update fails.
	UpdateBatch(batch []models.Metrics) error

	// GetGauge retrieves a gauge metric by name.
	// Parameters:
	//   - name: metric identifier
	// Returns the float64 value or error if metric doesn't exist or is not a gauge.
	GetGauge(name string) (float64, error)

	// GetCounter retrieves a counter metric by name.
	// Parameters:
	//   - name: metric identifier
	// Returns the int64 value or error if metric doesn't exist or is not a counter.
	GetCounter(name string) (int64, error)

	// GetAll returns all stored metrics as a map.
	// Key: metric ID, Value: pointer to Metrics struct.
	// Returns error if retrieval fails.
	GetAll() (map[string]*models.Metrics, error)
}

// MemStorage is an in-memory implementation of Storage interface.
// It stores metrics in a thread-safe map and optionally persists them to a file.
// MemStorage supports:
//   - Gauge metrics: floating-point values that can be set to any value
//   - Counter metrics: integer values that accumulate (add to existing)
//   - Automatic periodic persistence to JSON file
//   - Synchronous persistence option (storeInterval = 0)
type MemStorage struct {
	mu        *sync.Mutex
	metrics   map[string]*models.Metrics
	filePath  string
	syncWrite bool
}

// NewMemStorage creates a new in-memory storage backend for metrics.
// Parameters:
//   - filePath: Path to JSON file for persistence (empty string = no persistence)
//   - storeInterval: Interval for automatic saves (0 = sync on every update)
//
// Returns a fully initialized MemStorage ready for use.
// If filePath is non-empty and the file exists, metrics will be loaded from it on next access.
// If storeInterval is 0, metrics are persisted synchronously after each update.
// If storeInterval > 0, metrics are persisted periodically (StartAutoSave must be called).
func NewMemStorage(filePath string, storeInterval time.Duration) *MemStorage {
	return &MemStorage{
		mu:        &sync.Mutex{},
		metrics:   make(map[string]*models.Metrics),
		filePath:  filePath,
		syncWrite: storeInterval == 0,
	}
}

// Update updates or creates a metric in memory by its type and name.
//
// If the metric type is unsupported or the value cannot be parsed,
// an error is returned.
//
// When syncWrite is enabled and filePath is set, the storage is
// automatically persisted to disk after the update.
func (s *MemStorage) Update(metricType, name, value string) error {
	switch metricType {
	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}

		s.mu.Lock()
		s.metrics[name] = &models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		}
		s.mu.Unlock()

	case models.Counter:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}

		s.mu.Lock()
		if existing, ok := s.metrics[name]; ok && existing.Delta != nil {
			v += *existing.Delta
		}
		s.metrics[name] = &models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &v,
		}
		s.mu.Unlock()

	default:
		return errors.New("unsupported metric type")
	}

	if s.syncWrite && s.filePath != "" {
		_ = s.SaveToFile(s.filePath)
	}

	return nil
}

// GetGauge retrieves a gauge metric by name.
// Returns the float64 value if the metric exists.
// Returns an error if the metric does not exist or is not a gauge.
func (s *MemStorage) GetGauge(name string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.metrics[name]
	if !ok || m.Value == nil {
		return 0, errors.New("gauge not found")
	}
	return *m.Value, nil
}

// GetCounter retrieves a counter metric by name.
// Returns the int64 value if the metric exists.
// Returns an error if the metric does not exist or is not a counter.
func (s *MemStorage) GetCounter(name string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.metrics[name]
	if !ok || m.Delta == nil {
		return 0, errors.New("counter not found")
	}
	return *m.Delta, nil
}

// GetAll returns a copy of all stored metrics.
// Each metric is copied to prevent external modification of internal storage.
// Returns a map where the key is the metric ID and the value is a pointer to Metrics.
func (s *MemStorage) GetAll() (map[string]*models.Metrics, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]*models.Metrics, len(s.metrics))
	for k, v := range s.metrics {
		copied := *v
		result[k] = &copied
	}
	return result, nil
}

// SaveToFile persists all metrics to a JSON file.
// The file is created/overwritten with all current metrics.
// This method is thread-safe and acquires the storage mutex.
//
// Parameters:
//   - filePath: Path where metrics will be saved
//
// Returns error if file cannot be created or written to.
func (s *MemStorage) SaveToFile(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := make([]*models.Metrics, 0, len(s.metrics))
	for _, m := range s.metrics {
		copied := *m
		data = append(data, &copied)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		cerr := file.Close()
		if err == nil && cerr != nil {
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()
	return json.NewEncoder(file).Encode(data)
}

// LoadFromFile loads metrics from a JSON file into storage.
// If the file doesn't exist, returns nil (no error).
// This method is thread-safe and acquires the storage mutex.
//
// Parameters:
//   - filePath: Path to the JSON file containing metrics
//
// Returns error if file exists but cannot be read, or JSON is invalid.
// Returns nil if file doesn't exist (common on first startup).
func (s *MemStorage) LoadFromFile(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	var data []*models.Metrics
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return err
	}

	for _, m := range data {
		s.metrics[m.ID] = m
	}
	return nil
}

// StartAutoSave starts a background goroutine that periodically saves metrics to file.
// If interval <= 0, saves immediately and returns without starting background task.
// If interval > 0, creates a ticker that saves metrics at the specified interval.
// The goroutine continues until stopCh is closed.
//
// Parameters:
//   - filePath: Path where metrics will be saved
//   - interval: Duration between saves (0 = save immediately and stop)
//   - stopCh: Channel to signal goroutine to stop
//
// Typical usage:
//
//	stopCh := make(chan struct{})
//	defer close(stopCh)
//	storage.StartAutoSave("metrics.json", 15*time.Second, stopCh)
func (s *MemStorage) StartAutoSave(filePath string, interval time.Duration, stopCh <-chan struct{}) {
	if interval <= 0 {
		_ = s.SaveToFile(filePath)
		return
	}

	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				_ = s.SaveToFile(filePath)
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

// UpdateBatch updates multiple metrics at once.
// For gauge metrics: replaces the value with the last value in the batch.
// For counter metrics: adds deltas to existing values.
// Invalid metrics (empty ID or unknown type) are skipped.
// If MemStorage is in synchronous mode and filePath is set, changes are persisted immediately.
func (s *MemStorage) UpdateBatch(batch []models.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range batch {
		if m.ID == "" || (m.MType != models.Gauge && m.MType != models.Counter) {
			continue
		}

		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				continue
			}
			s.metrics[m.ID] = &models.Metrics{
				ID:    m.ID,
				MType: models.Gauge,
				Value: m.Value,
			}

		case models.Counter:
			if m.Delta == nil {
				continue
			}
			newDelta := *m.Delta
			if ex, ok := s.metrics[m.ID]; ok && ex.Delta != nil {
				newDelta += *ex.Delta
			}
			s.metrics[m.ID] = &models.Metrics{
				ID:    m.ID,
				MType: models.Counter,
				Delta: &newDelta,
			}
		}
	}

	if s.syncWrite && s.filePath != "" {
		return s.SaveToFile(s.filePath)
	}
	return nil
}

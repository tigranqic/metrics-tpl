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

type Storage interface {
	Update(metricType, name, value string) error
	UpdateBatch(batch []models.Metrics) error
	GetGauge(name string) (float64, error)
	GetCounter(name string) (int64, error)
	GetAll() (map[string]*models.Metrics, error)
}

type MemStorage struct {
	mu        *sync.Mutex
	metrics   map[string]*models.Metrics
	filePath  string
	syncWrite bool
}

func NewMemStorage(filePath string, storeInterval time.Duration) *MemStorage {
	return &MemStorage{
		mu:        &sync.Mutex{},
		metrics:   make(map[string]*models.Metrics),
		filePath:  filePath,
		syncWrite: storeInterval == 0,
	}
}

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

func (s *MemStorage) GetGauge(name string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.metrics[name]
	if !ok || m.Value == nil {
		return 0, errors.New("gauge not found")
	}
	return *m.Value, nil
}

func (s *MemStorage) GetCounter(name string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.metrics[name]
	if !ok || m.Delta == nil {
		return 0, errors.New("counter not found")
	}
	return *m.Delta, nil
}

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

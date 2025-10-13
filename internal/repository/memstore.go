package repository

import (
	"errors"
	"strconv"
	"sync"

	models "github.com/tigranqic/metrics-tpl/internal/model"
)

type Storage interface {
	Update(metricType, name, value string) error
	GetGauge(name string) (float64, error)
	GetCounter(name string) (int64, error)
}

type MemStorage struct {
	mu      sync.RWMutex
	metrics map[string]*models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]*models.Metrics),
	}
}

func (s *MemStorage) Update(metricType, name, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch metricType {
	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		s.metrics[name] = &models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		}
	case models.Counter:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		if existing, ok := s.metrics[name]; ok && existing.Delta != nil {
			v += *existing.Delta
		}
		s.metrics[name] = &models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &v,
		}
	default:
		return errors.New("unsupported metric type")
	}
	return nil
}

func (s *MemStorage) GetGauge(name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.metrics[name]
	if !ok || m.Value == nil {
		return 0, errors.New("gauge not found")
	}
	return *m.Value, nil
}

func (s *MemStorage) GetCounter(name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.metrics[name]
	if !ok || m.Delta == nil {
		return 0, errors.New("counter not found")
	}
	return *m.Delta, nil
}

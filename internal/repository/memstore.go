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
	GetAll() map[string]*models.Metrics
}

type MemStorage struct {
	mu      *sync.Mutex
	metrics map[string]*models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		mu:      &sync.Mutex{},
		metrics: make(map[string]*models.Metrics),
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

func (s *MemStorage) GetAll() map[string]*models.Metrics {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]*models.Metrics, len(s.metrics))
	for k, v := range s.metrics {
		copied := *v
		result[k] = &copied
	}
	return result
}

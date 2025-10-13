package repository

import (
	"errors"
	"strconv"
	"sync"

	models "github.com/tigranqic/metrics-tpl/internal/model"
)

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

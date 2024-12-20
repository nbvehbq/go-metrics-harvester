// Memory storage - implementation of Repository interface
package memory

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage"
)

// Storage - хранилище метрик
type Storage struct {
	mu      *sync.RWMutex
	storage map[string]metric.Metric
}

// NewFrom - creates a new memory storage from io.Reader interface
func NewFrom(src io.Reader) (*Storage, error) {
	var list []metric.Metric
	if err := json.NewDecoder(src).Decode(&list); err != nil {
		return nil, err
	}

	s := make(map[string]metric.Metric, len(list))
	for _, m := range list {
		s[m.ID] = m
	}

	return &Storage{storage: s, mu: &sync.RWMutex{}}, nil
}

// NewMemStorage - конструктор для хранения метрик
func NewMemStorage() *Storage {
	s := make(map[string]metric.Metric)

	return &Storage{storage: s, mu: &sync.RWMutex{}}
}

// Persist - save metrics to io.Writer
func (s *Storage) Persist(_ context.Context, dest io.Writer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var list []metric.Metric
	for _, v := range s.storage {
		list = append(list, v)
	}

	if err := json.NewEncoder(dest).Encode(&list); err != nil {
		return err
	}

	return nil
}

// Set - update or rewrite metric depends on metric type
func (s *Storage) Set(_ context.Context, value metric.Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if value.MType == metric.Counter && value.Delta == nil {
		return storage.ErrMetricMalformed
	}

	if value.MType == metric.Gauge && value.Value == nil {
		return storage.ErrMetricMalformed
	}

	v, ok := s.storage[value.ID]
	if !ok {
		s.storage[value.ID] = value
		return nil
	}

	switch value.MType {
	case metric.Gauge:
		val := *value.Value
		v.Value = &val
	case metric.Counter:
		*v.Delta += *value.Delta
	}

	s.storage[value.ID] = v
	return nil
}

// Get - get metric
func (s *Storage) Get(_ context.Context, key string) (metric.Metric, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.storage[key]
	return v, ok
}

// List - get all metrics
func (s *Storage) List(_ context.Context) ([]metric.Metric, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var list []metric.Metric
	for _, v := range s.storage {
		list = append(list, v)
	}

	return list, nil
}

func (s *Storage) Ping(_ context.Context) error {
	return storage.ErrNotSupported
}

// Update - update metrics with new values
func (s *Storage) Update(_ context.Context, m []metric.Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, value := range m {
		v, ok := s.storage[value.ID]
		if !ok {
			s.storage[value.ID] = value
			continue
		}

		switch value.MType {
		case metric.Gauge:
			val := *value.Value
			v.Value = &val
		case metric.Counter:
			*v.Delta += *value.Delta
		}

		s.storage[value.ID] = v
	}

	return nil
}

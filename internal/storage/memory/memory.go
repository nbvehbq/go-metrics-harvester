package memory

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage"
)

type Storage struct {
	mu      *sync.Mutex
	storage map[string]metric.Metric
}

func NewFrom(src io.Reader) (*Storage, error) {
	var list []metric.Metric
	if err := json.NewDecoder(src).Decode(&list); err != nil {
		return nil, err
	}

	s := make(map[string]metric.Metric, len(list))
	for _, m := range list {
		s[m.ID] = m
	}

	return &Storage{storage: s, mu: &sync.Mutex{}}, nil
}

func NewMemStorage() *Storage {
	s := make(map[string]metric.Metric)

	return &Storage{storage: s, mu: &sync.Mutex{}}
}

func (s *Storage) Persist(dest io.Writer) error {
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

func (s *Storage) Set(value metric.Metric) error {
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

func (s *Storage) Get(key string) (metric.Metric, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.storage[key]
	return v, ok
}

func (s *Storage) List() ([]metric.Metric, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var list []metric.Metric
	for _, v := range s.storage {
		list = append(list, v)
	}

	return list, nil
}

func (s *Storage) Ping(ctx context.Context) error {
	return storage.ErrNotSupported
}

func (s *Storage) Update(ctx context.Context, m []metric.Metric) error {
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

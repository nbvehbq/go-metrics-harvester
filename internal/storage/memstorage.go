package storage

import (
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
)

type MemStorage struct {
	mu      *sync.Mutex
	storage map[string]metric.Metric
}

func NewFrom(src io.Reader) (*MemStorage, error) {
	var list []metric.Metric
	if err := json.NewDecoder(src).Decode(&list); err != nil {
		if errors.Is(err, io.EOF) {
			logger.Log.Info("Storage file empty")
			return NewMemStorage(), nil
		}

		return nil, err
	}
	s := make(map[string]metric.Metric, len(list))
	for _, m := range list {
		s[m.ID] = m
	}

	return &MemStorage{storage: s, mu: &sync.Mutex{}}, nil
}

func NewMemStorage() *MemStorage {
	s := make(map[string]metric.Metric)

	return &MemStorage{storage: s, mu: &sync.Mutex{}}
}

func (m *MemStorage) Persist(dest io.Writer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []metric.Metric
	for _, v := range m.storage {
		list = append(list, v)
	}

	if err := json.NewEncoder(dest).Encode(&list); err != nil {
		return err
	}

	return nil
}

func (m *MemStorage) Set(value metric.Metric) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if value.MType == metric.Counter && value.Delta == nil {
		return
	}

	if value.MType == metric.Gauge && value.Value == nil {
		return
	}

	v, ok := m.storage[value.ID]
	if !ok {
		m.storage[value.ID] = value
		return
	}

	switch value.MType {
	case metric.Gauge:
		val := *value.Value
		v.Value = &val
	case metric.Counter:
		*v.Delta += *value.Delta
	}

	m.storage[value.ID] = v
}

func (m *MemStorage) Get(key string) (metric.Metric, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok := m.storage[key]
	return v, ok
}

func (m *MemStorage) List() []metric.Metric {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []metric.Metric
	for _, v := range m.storage {
		list = append(list, v)
	}

	return list
}

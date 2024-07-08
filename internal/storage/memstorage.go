package storage

import (
	"strconv"
	"sync"

	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
)

type MemStorage struct {
	mu      *sync.Mutex
	storage map[string]metric.Metric
}

func NewMemStorage() *MemStorage {
	s := make(map[string]metric.Metric)

	return &MemStorage{storage: s, mu: &sync.Mutex{}}
}

func (m *MemStorage) Set(value metric.Metric) metric.Metric {
	m.mu.Lock()
	v, ok := m.storage[value.Name]
	v.Type = value.Type
	v.Name = value.Name

	switch value.Type {
	case metric.Gauge:
		f, _ := strconv.ParseFloat(value.Value.(string), 64)
		v.Value = f
	case metric.Counter:
		i, _ := strconv.Atoi(value.Value.(string))
		if ok {
			v.Value = v.Value.(int64) + int64(i)
		} else {
			v.Value = int64(i)
		}
	}

	m.storage[value.Name] = v
	m.mu.Unlock()

	return v
}

func (m *MemStorage) Get(key string) (metric.Metric, bool) {
	v, ok := m.storage[key]
	return v, ok
}

func (m *MemStorage) List() []metric.Metric {
	var list []metric.Metric
	for _, v := range m.storage {
		list = append(list, v)
	}

	return list
}

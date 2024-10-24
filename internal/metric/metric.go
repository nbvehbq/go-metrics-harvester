package metric

import (
	"strconv"
	"sync"
)

const (
	Gauge   = "gauge"
	Counter = "counter"
)

type Validator func(value string) bool

func validateCounter(value string) bool {
	_, err := strconv.ParseInt(value, 10, 64)
	return err == nil
}

func validateGauge(value string) bool {
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

var (
	AllowedMetricType = map[string]Validator{
		Gauge:   validateGauge,
		Counter: validateCounter,
	}
)

// Metric - структура для хранения метрик
type Metric struct {
	ID    string   `json:"id" db:"id"`                 // имя метрики
	MType string   `json:"type" db:"mtype"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty" db:"delta"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty" db:"value"` // значение метрики в случае передачи gauge
}

// Metrics - хранилище метрик
type Metrics struct {
	Mu      *sync.RWMutex
	Metrics map[string]Metric
}

// NewMetrics - конструктор для хранения метрик
func NewMetrics() *Metrics {
	return &Metrics{
		Mu:      &sync.RWMutex{},
		Metrics: make(map[string]Metric),
	}
}

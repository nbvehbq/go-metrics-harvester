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
	_, err := strconv.Atoi(value)
	return err == nil
}

func validateGauge(value string) bool {
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

var (
	AllowedMetricName = map[string]Validator{
		Gauge:   validateGauge,
		Counter: validateCounter,
	}
)

type Metric struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

type Metrics struct {
	Mu      *sync.Mutex
	Metrics map[string]Metric
}

func NewMetrics() *Metrics {
	return &Metrics{
		Mu:      &sync.Mutex{},
		Metrics: make(map[string]Metric),
	}
}

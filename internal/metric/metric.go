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
	Name  string
	Type  string
	Value interface{}
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

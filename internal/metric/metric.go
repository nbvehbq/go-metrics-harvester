package metric

import (
	"strconv"
	"sync"
)

const (
	Gauge   = "gauge"
	Counter = "counter"
)

type Validator func(value interface{}) bool

func validateCounter(value interface{}) bool {
	_, err := strconv.Atoi(value.(string))
	return err == nil
}

func validateGauge(value interface{}) bool {
	_, err := strconv.ParseFloat(value.(string), 64)
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
		Mu: &sync.Mutex{},
		Metrics: map[string]Metric{
			"Alloc":         {Name: "Alloc", Type: Gauge, Value: nil},
			"BuckHashSys":   {Name: "BuckHashSys", Type: Gauge, Value: nil},
			"Frees":         {Name: "Frees", Type: Gauge, Value: nil},
			"GCCPUFraction": {Name: "GCCPUFraction", Type: Gauge, Value: nil},
			"GCSys":         {Name: "GCSys", Type: Gauge, Value: nil},
			"HeapAlloc":     {Name: "HeapAlloc", Type: Gauge, Value: nil},
			"HeapIdle":      {Name: "HeapIdle", Type: Gauge, Value: nil},
			"HeapInuse":     {Name: "HeapInuse", Type: Gauge, Value: nil},
			"HeapObjects":   {Name: "HeapObjects", Type: Gauge, Value: nil},
			"HeapReleased":  {Name: "HeapReleased", Type: Gauge, Value: nil},
			"HeapSys":       {Name: "HeapSys", Type: Gauge, Value: nil},
			"LastGC":        {Name: "LastGC", Type: Gauge, Value: nil},
			"Lookups":       {Name: "Lookups", Type: Gauge, Value: nil},
			"MCacheInuse":   {Name: "MCacheInuse", Type: Gauge, Value: nil},
			"MCacheSys":     {Name: "MCacheSys", Type: Gauge, Value: nil},
			"MSpanInuse":    {Name: "MSpanInuse", Type: Gauge, Value: nil},
			"MSpanSys":      {Name: "MSpanSys", Type: Gauge, Value: nil},
			"Mallocs":       {Name: "Mallocs", Type: Gauge, Value: nil},
			"NextGC":        {Name: "NextGC", Type: Gauge, Value: nil},
			"NumForcedGC":   {Name: "NumForcedGC", Type: Gauge, Value: nil},
			"NumGC":         {Name: "NumGC", Type: Gauge, Value: nil},
			"OtherSys":      {Name: "OtherSys", Type: Gauge, Value: nil},
			"PauseTotalNs":  {Name: "PauseTotalNs", Type: Gauge, Value: nil},
			"StackInuse":    {Name: "StackInuse", Type: Gauge, Value: nil},
			"StackSys":      {Name: "StackSys", Type: Gauge, Value: nil},
			"Sys":           {Name: "Sys", Type: Gauge, Value: nil},
			"TotalAlloc":    {Name: "TotalAlloc", Type: Gauge, Value: nil},
			"PollCount":     {Name: "PollCount", Type: Counter, Value: nil},
			"RandomValue":   {Name: "RandomValue", Type: Gauge, Value: nil},
		},
	}
}

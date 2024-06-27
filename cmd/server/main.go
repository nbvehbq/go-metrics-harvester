package main

import (
	"net/http"
	"strconv"
	"strings"
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
	allowedMetricName = map[string]Validator{
		"gauge":   validateGauge,
		"counter": validateCounter,
	}
	storage = NewMemStorage()
)

type Metric struct {
	Type  string
	Value interface{}
}

type MemStorage struct {
	metrics map[string]Metric
}

func NewMemStorage() *MemStorage {
	s := make(map[string]Metric)

	return &MemStorage{metrics: s}
}

func (m *MemStorage) Set(key string, value Metric) Metric {
	v, ok := m.metrics[key]
	v.Type = value.Type

	switch value.Type {
	case "gauge":
		f, _ := strconv.ParseFloat(value.Value.(string), 64)
		v.Value = f
	case "counter":
		i, _ := strconv.Atoi(value.Value.(string))
		if ok {
			v.Value = v.Value.(int64) + int64(i)
		} else {
			v.Value = int64(i)
		}
	}
	m.metrics[key] = v

	return v
}

func (m *MemStorage) Get(key string) (Metric, bool) {
	v, ok := m.metrics[key]
	return v, ok
}

type Storage interface {
	Set(value Metric) Metric
	Get(key string) (Metric, bool)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/update/`, updateHandler)

	if err := http.ListenAndServe(`:8080`, mux); err != nil {
		panic(err)
	}
}

func updateHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	// check params
	parts := strings.Split(req.URL.Path, "/")
	if len(parts) != 5 {
		http.Error(res, "not found", http.StatusNotFound)
		return
	}

	// check metric type
	validate, ok := allowedMetricName[parts[2]]
	if !ok {
		http.Error(res, "bad request (type)", http.StatusBadRequest)
		return
	}

	// check metric value
	if !validate(parts[4]) {
		http.Error(res, "bad request (value)", http.StatusBadRequest)
		return
	}

	storage.Set(parts[3], Metric{Type: parts[2], Value: parts[4]})

	res.Header().Set("Content-Type", "plan/text")
}

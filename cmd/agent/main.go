package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"golang.org/x/sync/errgroup"
)

const (
	pollInterval   = time.Second * 2
	reportInterval = time.Second * 10
)

func main() {
	ctx := context.Background()
	runner, _ := errgroup.WithContext(ctx)

	metrics := metric.NewMetrics()

	runner.Go(func() error {
		for {
			requestMetrics(metrics)

			time.Sleep(pollInterval)
		}
	})

	runner.Go(func() error {
		for {
			if err := publishMetrics(runner, metrics); err != nil {
				return err
			}

			time.Sleep(reportInterval)
		}
	})

	runner.Wait()
}

func requestMetrics(m *metric.Metrics) {
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)

	m.Mu.Lock()

	m.Metrics["Alloc"] = metric.Metric{Name: "Alloc", Type: metric.Gauge, Value: float64(rtm.Alloc)}
	m.Metrics["BuckHashSys"] = metric.Metric{Name: "BuckHashSys", Type: metric.Gauge, Value: float64(rtm.BuckHashSys)}
	m.Metrics["Frees"] = metric.Metric{Name: "Frees", Type: metric.Gauge, Value: float64(rtm.Frees)}
	m.Metrics["GCCPUFraction"] = metric.Metric{Name: "GCCPUFraction", Type: metric.Gauge, Value: float64(rtm.GCCPUFraction)}
	m.Metrics["GCSys"] = metric.Metric{Name: "GCSys", Type: metric.Gauge, Value: float64(rtm.GCSys)}
	m.Metrics["HeapAlloc"] = metric.Metric{Name: "HeapAlloc", Type: metric.Gauge, Value: float64(rtm.HeapAlloc)}
	m.Metrics["HeapIdle"] = metric.Metric{Name: "HeapIdle", Type: metric.Gauge, Value: float64(rtm.HeapIdle)}
	m.Metrics["HeapInuse"] = metric.Metric{Name: "HeapInuse", Type: metric.Gauge, Value: float64(rtm.HeapInuse)}
	m.Metrics["HeapObjects"] = metric.Metric{Name: "HeapObjects", Type: metric.Gauge, Value: float64(rtm.HeapObjects)}
	m.Metrics["HeapReleased"] = metric.Metric{Name: "HeapReleased", Type: metric.Gauge, Value: float64(rtm.HeapReleased)}
	m.Metrics["HeapSys"] = metric.Metric{Name: "HeapSys", Type: metric.Gauge, Value: float64(rtm.HeapSys)}
	m.Metrics["LastGC"] = metric.Metric{Name: "LastGC", Type: metric.Gauge, Value: float64(rtm.LastGC)}
	m.Metrics["Lookups"] = metric.Metric{Name: "Lookups", Type: metric.Gauge, Value: float64(rtm.Lookups)}
	m.Metrics["MCacheInuse"] = metric.Metric{Name: "MCacheInuse", Type: metric.Gauge, Value: float64(rtm.MCacheInuse)}
	m.Metrics["MCacheSys"] = metric.Metric{Name: "MCacheSys", Type: metric.Gauge, Value: float64(rtm.MCacheSys)}
	m.Metrics["MSpanInuse"] = metric.Metric{Name: "MSpanInuse", Type: metric.Gauge, Value: float64(rtm.MSpanInuse)}
	m.Metrics["MSpanSys"] = metric.Metric{Name: "MSpanSys", Type: metric.Gauge, Value: float64(rtm.MSpanSys)}
	m.Metrics["Mallocs"] = metric.Metric{Name: "Mallocs", Type: metric.Gauge, Value: float64(rtm.Mallocs)}
	m.Metrics["NextGC"] = metric.Metric{Name: "NextGC", Type: metric.Gauge, Value: float64(rtm.NextGC)}
	m.Metrics["NumForcedGC"] = metric.Metric{Name: "NumForcedGC", Type: metric.Gauge, Value: float64(rtm.NumForcedGC)}
	m.Metrics["NumGC"] = metric.Metric{Name: "NumGC", Type: metric.Gauge, Value: float64(rtm.NumGC)}
	m.Metrics["OtherSys"] = metric.Metric{Name: "OtherSys", Type: metric.Gauge, Value: float64(rtm.OtherSys)}
	m.Metrics["PauseTotalNs"] = metric.Metric{Name: "PauseTotalNs", Type: metric.Gauge, Value: float64(rtm.PauseTotalNs)}
	m.Metrics["StackInuse"] = metric.Metric{Name: "StackInuse", Type: metric.Gauge, Value: float64(rtm.StackInuse)}
	m.Metrics["StackSys"] = metric.Metric{Name: "StackSys", Type: metric.Gauge, Value: float64(rtm.StackSys)}
	m.Metrics["Sys"] = metric.Metric{Name: "Sys", Type: metric.Gauge, Value: float64(rtm.Sys)}
	m.Metrics["TotalAlloc"] = metric.Metric{Name: "TotalAlloc", Type: metric.Gauge, Value: float64(rtm.TotalAlloc)}

	m.Metrics["PollCount"] = metric.Metric{Name: "PollCount", Type: metric.Counter, Value: 1}
	m.Metrics["RandomValue"] = metric.Metric{Name: "RandomValue", Type: metric.Gauge, Value: rand.Float64()}

	m.Mu.Unlock()

	log.Println("Metric requested")
}

func publishMetrics(runner *errgroup.Group, m *metric.Metrics) error {
	client := resty.New()

	m.Mu.Lock()
	for _, v := range m.Metrics {
		v := v
		runner.Go(func() error {
			if v.Value != nil {
				if err := makePostRequest(client, v); err != nil {
					log.Println("request error:", err)
					return err
				}
				return nil
			}
			return nil
		})
	}
	m.Mu.Unlock()

	log.Println("Metrics published")
	return nil
}

func makePostRequest(client *resty.Client, m metric.Metric) error {
	var value string
		switch m.Type {
		case metric.Counter:
			value = fmt.Sprintf("%d", m.Value)
		case metric.Gauge:
			value = fmt.Sprintf("%f", m.Value)
		}

	res, err := client.R().
	SetHeader("Content-Type", "text/plain").
	SetPathParams(map[string]string{
		"type": m.Type,
		"name": m.Name,
		"value": value,
	}).
	Post("http://localhost:8080/update/{type}/{name}/{value}")

	if err != nil {
		return err
	}

	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("status: %d", res.StatusCode())
	}

	return nil
}

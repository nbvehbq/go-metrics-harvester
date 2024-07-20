package agent

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Agent struct {
	cfg    *Config
	runner *errgroup.Group
	client *resty.Client
}

func NewAgent(r *errgroup.Group, cfg *Config) *Agent {
	c := resty.New()
	return &Agent{runner: r, cfg: cfg, client: c}
}

func (a *Agent) Run(ctx context.Context) {
	logger.Log.Info("Agent started.")
	metrics := metric.NewMetrics()

	a.runner.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second * time.Duration(a.cfg.PollInterval)):
				requestMetrics(metrics)
			}
		}
	})

	a.runner.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second * time.Duration(a.cfg.ReportInterval)):
				if err := a.publishMetrics(metrics); err != nil {
					return err
				}
			}
		}
	})
}

func floatPtr(val float64) *float64 {
	return &val
}

func intPtr(val int64) *int64 {
	return &val
}

func requestMetrics(m *metric.Metrics) {
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)

	m.Mu.Lock()

	m.Metrics["Alloc"] = metric.Metric{ID: "Alloc", MType: metric.Gauge, Value: floatPtr(float64(rtm.Alloc))}
	m.Metrics["BuckHashSys"] = metric.Metric{ID: "BuckHashSys", MType: metric.Gauge, Value: floatPtr(float64(rtm.BuckHashSys))}
	m.Metrics["Frees"] = metric.Metric{ID: "Frees", MType: metric.Gauge, Value: floatPtr(float64(rtm.Frees))}
	m.Metrics["GCCPUFraction"] = metric.Metric{ID: "GCCPUFraction", MType: metric.Gauge, Value: floatPtr(float64(rtm.GCCPUFraction))}
	m.Metrics["GCSys"] = metric.Metric{ID: "GCSys", MType: metric.Gauge, Value: floatPtr(float64(rtm.GCSys))}
	m.Metrics["HeapAlloc"] = metric.Metric{ID: "HeapAlloc", MType: metric.Gauge, Value: floatPtr(float64(rtm.HeapAlloc))}
	m.Metrics["HeapIdle"] = metric.Metric{ID: "HeapIdle", MType: metric.Gauge, Value: floatPtr(float64(rtm.HeapIdle))}
	m.Metrics["HeapInuse"] = metric.Metric{ID: "HeapInuse", MType: metric.Gauge, Value: floatPtr(float64(rtm.HeapInuse))}
	m.Metrics["HeapObjects"] = metric.Metric{ID: "HeapObjects", MType: metric.Gauge, Value: floatPtr(float64(rtm.HeapObjects))}
	m.Metrics["HeapReleased"] = metric.Metric{ID: "HeapReleased", MType: metric.Gauge, Value: floatPtr(float64(rtm.HeapReleased))}
	m.Metrics["HeapSys"] = metric.Metric{ID: "HeapSys", MType: metric.Gauge, Value: floatPtr(float64(rtm.HeapSys))}
	m.Metrics["LastGC"] = metric.Metric{ID: "LastGC", MType: metric.Gauge, Value: floatPtr(float64(rtm.LastGC))}
	m.Metrics["Lookups"] = metric.Metric{ID: "Lookups", MType: metric.Gauge, Value: floatPtr(float64(rtm.Lookups))}
	m.Metrics["MCacheInuse"] = metric.Metric{ID: "MCacheInuse", MType: metric.Gauge, Value: floatPtr(float64(rtm.MCacheInuse))}
	m.Metrics["MCacheSys"] = metric.Metric{ID: "MCacheSys", MType: metric.Gauge, Value: floatPtr(float64(rtm.MCacheSys))}
	m.Metrics["MSpanInuse"] = metric.Metric{ID: "MSpanInuse", MType: metric.Gauge, Value: floatPtr(float64(rtm.MSpanInuse))}
	m.Metrics["MSpanSys"] = metric.Metric{ID: "MSpanSys", MType: metric.Gauge, Value: floatPtr(float64(rtm.MSpanSys))}
	m.Metrics["Mallocs"] = metric.Metric{ID: "Mallocs", MType: metric.Gauge, Value: floatPtr(float64(rtm.Mallocs))}
	m.Metrics["NextGC"] = metric.Metric{ID: "NextGC", MType: metric.Gauge, Value: floatPtr(float64(rtm.NextGC))}
	m.Metrics["NumForcedGC"] = metric.Metric{ID: "NumForcedGC", MType: metric.Gauge, Value: floatPtr(float64(rtm.NumForcedGC))}
	m.Metrics["NumGC"] = metric.Metric{ID: "NumGC", MType: metric.Gauge, Value: floatPtr(float64(rtm.NumGC))}
	m.Metrics["OtherSys"] = metric.Metric{ID: "OtherSys", MType: metric.Gauge, Value: floatPtr(float64(rtm.OtherSys))}
	m.Metrics["PauseTotalNs"] = metric.Metric{ID: "PauseTotalNs", MType: metric.Gauge, Value: floatPtr(float64(rtm.PauseTotalNs))}
	m.Metrics["StackInuse"] = metric.Metric{ID: "StackInuse", MType: metric.Gauge, Value: floatPtr(float64(rtm.StackInuse))}
	m.Metrics["StackSys"] = metric.Metric{ID: "StackSys", MType: metric.Gauge, Value: floatPtr(float64(rtm.StackSys))}
	m.Metrics["Sys"] = metric.Metric{ID: "Sys", MType: metric.Gauge, Value: floatPtr(float64(rtm.Sys))}
	m.Metrics["TotalAlloc"] = metric.Metric{ID: "TotalAlloc", MType: metric.Gauge, Value: floatPtr(float64(rtm.TotalAlloc))}

	m.Metrics["PollCount"] = metric.Metric{ID: "PollCount", MType: metric.Counter, Delta: intPtr(1)}
	m.Metrics["RandomValue"] = metric.Metric{ID: "RandomValue", MType: metric.Gauge, Value: floatPtr(rand.Float64())}

	m.Mu.Unlock()

	logger.Log.Info("Metric requested")
}

func (a *Agent) publishMetrics(m *metric.Metrics) error {
	m.Mu.Lock()
	defer func() {
		m.Mu.Unlock()
		logger.Log.Info("Metrics published")
	}()

	for _, v := range m.Metrics {
		v := v
		a.runner.Go(func() error {
			if err := a.makePostRequest(v); err != nil {
				logger.Log.Error("request error:", zap.Error(err))
				return nil
			}
			return nil
		})
	}

	return nil
}

func (a *Agent) makePostRequest(m metric.Metric) error {
	res, err := a.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&m).
		Post(fmt.Sprintf("%s/update/", a.cfg.Address))

	if err != nil {
		return err
	}

	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("status: %d", res.StatusCode())
	}

	return nil
}

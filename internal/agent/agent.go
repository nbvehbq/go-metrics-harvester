package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/nbvehbq/go-metrics-harvester/internal/hash"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/retry"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
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
				a.runner.Go(func() error {
					if err := requestMemoryMetrics(ctx, metrics); err != nil {
						logger.Log.Error("requestMemoryMetrics", zap.Error(err))
					}
					return nil
				})

				a.runner.Go(func() error {
					requestMetrics(metrics)
					return nil
				})
			}
		}
	})

	a.runner.Go(func() error {
		const numJobs = 1024

		jobs := make(chan *metric.Metrics, numJobs)
		defer close(jobs)

		results := make(chan error, numJobs)

		for range a.cfg.RateLimit {
			a.runner.Go(func() error {
				return a.worker(ctx, jobs, results)
			})
		}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second * time.Duration(a.cfg.ReportInterval)):
				jobs <- metrics
			case err := <-results:
				if err != nil {
					logger.Log.Error("publish", zap.Error(err))
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

func requestMemoryMetrics(ctx context.Context, m *metric.Metrics) error {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return errors.Wrap(err, "memory")
	}
	percents, err := cpu.PercentWithContext(ctx, 0, true)
	if err != nil {
		return errors.Wrap(err, "cpu")
	}

	m.Metrics["TotalMemory"] = metric.Metric{ID: "TotalMemory", MType: metric.Gauge, Value: floatPtr(float64(v.Total))}
	m.Metrics["FreeMemory"] = metric.Metric{ID: "FreeMemory", MType: metric.Gauge, Value: floatPtr(float64(v.Free))}

	for i, p := range percents {
		ID := fmt.Sprintf("CPUutilization%d", i+1)
		m.Metrics[ID] = metric.Metric{ID: ID, MType: metric.Gauge, Value: floatPtr(float64(p))}
	}

	return nil
}

func (a *Agent) publishMetrics(m *metric.Metrics) error {
	m.Mu.RLock()
	defer func() {
		m.Mu.RUnlock()
		logger.Log.Info("Metrics published")
	}()

	list := make([]metric.Metric, 0, len(m.Metrics))
	for _, v := range m.Metrics {
		list = append(list, v)
	}

	buf, err := json.Marshal(list)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}

	buf, err = compress(buf)
	if err != nil {
		return errors.Wrap(err, "compress")
	}

	var res *resty.Response
	err = retry.Do(func() (err error) {
		req := a.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept-Encoding", "gzip").
			SetHeader("Content-Encoding", "gzip")

		if a.cfg.Key != "" {
			sign := hash.Hash([]byte(a.cfg.Key), buf)
			req = req.SetHeader(hash.HashHeaderKey, base64.StdEncoding.EncodeToString(sign))
		}

		res, err = req.
			SetBody(buf).
			Post(fmt.Sprintf("%s/updates/", a.cfg.Address))

		return
	})

	if err != nil {
		return errors.Wrap(err, "resty post")
	}

	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("status: %d", res.StatusCode())
	}

	return nil
}

func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)

	_, err := w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to buffer: %v", err)
	}

	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %v", err)
	}

	return b.Bytes(), nil
}

func (a *Agent) worker(ctx context.Context, jobs <-chan *metric.Metrics, results chan<- error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case job, ok := <-jobs:
			if !ok {
				return nil
			}
			results <- a.publishMetrics(job)
			return nil
		}
	}
}

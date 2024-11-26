package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	xhash "hash"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/nbvehbq/go-metrics-harvester/internal/hash"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/pkg/retry"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Agent is a metrics harvester agent
type Agent struct {
	cfg       *Config
	runner    *errgroup.Group
	client    http.Client
	publicKey []byte
}

// NewAgent creates a new agent
func NewAgent(r *errgroup.Group, cfg *Config, client http.Client) (*Agent, error) {
	var (
		buf []byte
		err error
	)
	if cfg.CryptoKey != "" {
		buf, err = os.ReadFile(cfg.CryptoKey)
		if err != nil {
			return nil, errors.Wrap(err, "open public key filename")
		}
	}

	return &Agent{runner: r, cfg: cfg, client: client, publicKey: buf}, nil
}

// Run runs the agent
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

	if a.publicKey != nil {
		publicKeyBlock, _ := pem.Decode(a.publicKey)
		publicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
		if err != nil {
			return errors.Wrap(err, "parse public key")
		}

		buf, err = encryptOAEP(sha256.New(), crand.Reader, publicKey.(*rsa.PublicKey), buf, nil)
		if err != nil {
			return errors.Wrap(err, "encrypt body")
		}
	}

	buf, err = compress(buf)
	if err != nil {
		return errors.Wrap(err, "compress")
	}

	err = retry.Do(func() (err error) {
		req, err := http.NewRequest(
			"POST",
			fmt.Sprintf("%s/updates/", a.cfg.Address),
			bytes.NewReader(buf))
		if err != nil {
			return errors.Wrap(err, "new request")
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept-Encoding", "gzip")
		req.Header.Add("Content-Encoding", "gzip")

		addr, err := realIP()
		if err != nil {
			return errors.Wrap(err, "get ip address")
		}
		req.Header.Add("X-Real-IP", addr)

		if a.cfg.Key != "" {
			sign := hash.Hash([]byte(a.cfg.Key), buf)
			req.Header.Add(hash.HashHeaderKey, base64.StdEncoding.EncodeToString(sign))
		}

		res, err := a.client.Do(req)
		if err != nil {
			return errors.Wrap(err, "send request")
		}
		defer res.Body.Close()

		return
	})

	if err != nil {
		return errors.Wrap(err, "client post")
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

func encryptOAEP(hash xhash.Hash, random io.Reader, public *rsa.PublicKey, msg []byte, label []byte) ([]byte, error) {
	msgLen := len(msg)
	step := public.Size() - 2*hash.Size() - 2
	var encryptedBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		encryptedBlockBytes, err := rsa.EncryptOAEP(hash, random, public, msg[start:finish], label)
		if err != nil {
			return nil, err
		}

		encryptedBytes = append(encryptedBytes, encryptedBlockBytes...)
	}

	return encryptedBytes, nil
}

func realIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}

	return "127.0.0.1", nil
}

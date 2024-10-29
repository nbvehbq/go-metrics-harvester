package agent

import (
	"context"
	"testing"

	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/stretchr/testify/assert"
)

func Test_requestMetrics(t *testing.T) {
	type args struct {
		m *metric.Metrics
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "request metrics",
			args: args{m: metric.NewMetrics()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestMetrics(tt.args.m)
			assert.Contains(t, tt.args.m.Metrics, "Alloc")
			assert.Contains(t, tt.args.m.Metrics, "BuckHashSys")
			assert.Contains(t, tt.args.m.Metrics, "Frees")
			assert.Contains(t, tt.args.m.Metrics, "GCCPUFraction")
			assert.Contains(t, tt.args.m.Metrics, "GCSys")
			assert.Contains(t, tt.args.m.Metrics, "HeapAlloc")
			assert.Contains(t, tt.args.m.Metrics, "HeapIdle")
			assert.Contains(t, tt.args.m.Metrics, "HeapInuse")
			assert.Contains(t, tt.args.m.Metrics, "HeapObjects")
			assert.Contains(t, tt.args.m.Metrics, "HeapReleased")
			assert.Contains(t, tt.args.m.Metrics, "HeapSys")
			assert.Contains(t, tt.args.m.Metrics, "LastGC")
			assert.Contains(t, tt.args.m.Metrics, "Lookups")
			assert.Contains(t, tt.args.m.Metrics, "MCacheInuse")
			assert.Contains(t, tt.args.m.Metrics, "MCacheSys")
			assert.Contains(t, tt.args.m.Metrics, "MSpanInuse")
			assert.Contains(t, tt.args.m.Metrics, "MSpanSys")
			assert.Contains(t, tt.args.m.Metrics, "Mallocs")
			assert.Contains(t, tt.args.m.Metrics, "NextGC")
			assert.Contains(t, tt.args.m.Metrics, "NumForcedGC")
			assert.Contains(t, tt.args.m.Metrics, "NumGC")
			assert.Contains(t, tt.args.m.Metrics, "OtherSys")
			assert.Contains(t, tt.args.m.Metrics, "PauseTotalNs")
			assert.Contains(t, tt.args.m.Metrics, "StackInuse")
			assert.Contains(t, tt.args.m.Metrics, "StackSys")
			assert.Contains(t, tt.args.m.Metrics, "Sys")
			assert.Contains(t, tt.args.m.Metrics, "TotalAlloc")
		})
	}
}

func Test_requestMemoryMetrics(t *testing.T) {
	type args struct {
		m *metric.Metrics
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "request memory metrics",
			args: args{m: metric.NewMetrics()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requestMemoryMetrics(context.Background(), tt.args.m)
			assert.NoError(t, err)
			assert.Contains(t, tt.args.m.Metrics, "TotalMemory")
			assert.Contains(t, tt.args.m.Metrics, "FreeMemory")
		})
	}
}

func Test_commpress(t *testing.T) {
	payload := []byte("hello world")
	b, err := compress(payload)
	assert.NoError(t, err)
	assert.NotEqual(t, payload, b)
}

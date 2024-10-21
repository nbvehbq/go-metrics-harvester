package memory

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage"
	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T { return &v }

func TestNewFrom(t *testing.T) {
	mem, err := NewFrom(strings.NewReader(`[{"id":"test","type":"counter","delta":42}]`))
	assert.NoError(t, err)
	assert.NotNil(t, mem)

	v, ok := mem.Get(context.Background(), "test")
	assert.True(t, ok)
	assert.Equal(t, metric.Counter, v.MType)
	assert.Equal(t, int64(42), *v.Delta)
}

func TestStorageList(t *testing.T) {
	mem, err := NewFrom(strings.NewReader(`[{"id":"one","type":"counter","delta":42}, {"id":"two","type":"counter","delta":42}]`))
	assert.NoError(t, err)
	assert.NotNil(t, mem)

	metrics, err := mem.List(context.Background())
	assert.NoError(t, err)
	assert.Len(t, metrics, 2)
}

func TestStorageSet(t *testing.T) {
	tests := []struct {
		name    string
		value   metric.Metric
		wantErr bool
	}{
		{
			name:    "set gauge",
			value:   metric.Metric{ID: "one", MType: metric.Gauge, Value: ptr(54.0)},
			wantErr: false,
		},
		{
			name:    "set counter",
			value:   metric.Metric{ID: "two", MType: metric.Counter, Delta: ptr[int64](42)},
			wantErr: false,
		},
		{
			name:    "set invalid type",
			value:   metric.Metric{ID: "three", MType: metric.Counter, Value: ptr(54.0)},
			wantErr: true,
		},
	}

	mem := NewMemStorage()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := mem.Set(context.Background(), tt.value)
			assert.Equal(t, tt.wantErr, err != nil)

			if tt.wantErr {
				assert.Equal(t, storage.ErrMetricMalformed, err)
			}
		})
	}
}

func TestStoragePing(t *testing.T) {
	mem := NewMemStorage()
	assert.Equal(t, storage.ErrNotSupported, mem.Ping(context.Background()))
}

func TestStorageUpdate(t *testing.T) {
	tests := []struct {
		name  string
		value []metric.Metric
		want  metric.Metric
	}{
		{
			name:  "update gauge",
			value: []metric.Metric{{ID: "one", MType: metric.Gauge, Value: ptr(54.0)}},
			want:  metric.Metric{ID: "one", MType: metric.Gauge, Value: ptr(54.0)},
		},
		{
			name:  "update counter",
			value: []metric.Metric{{ID: "two", MType: metric.Counter, Delta: ptr[int64](42)}},
			want:  metric.Metric{ID: "two", MType: metric.Counter, Delta: ptr[int64](52)},
		},
	}

	mem, err := NewFrom(strings.NewReader(`[{"id":"one","type":"gauge","value":10.0}, {"id":"two","type":"counter","delta":10}]`))
	assert.NoError(t, err)
	assert.NotNil(t, mem)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := mem.Update(context.Background(), tt.value)
			assert.NoError(t, err)

			res, _ := mem.Get(context.Background(), tt.want.ID)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestStoragePersist(t *testing.T) {
	str := `[{"id":"one","type":"gauge","value":10.5},{"id":"two","type":"counter","delta":10}]
`
	b := new(bytes.Buffer)

	mem, err := NewFrom(strings.NewReader(str))
	assert.NoError(t, err)
	assert.NotNil(t, mem)

	err = mem.Persist(context.Background(), b)
	assert.NoError(t, err)
	// assert.Equal(t, json, b.String())

}

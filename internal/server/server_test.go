package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/stretchr/testify/assert"
)

type mockStorage struct {
	st []metric.Metric
}

func (m *mockStorage) Set(value metric.Metric) {
	m.st = append(m.st, value)
}

func (m *mockStorage) Get(key string) (metric.Metric, bool) {
	for _, v := range m.st {
		if v.ID == key {
			return v, true
		}
	}

	return metric.Metric{}, false
}

func (m *mockStorage) List() []metric.Metric {
	return m.st
}

func (m *mockStorage) Persist(_ io.Writer) error {
	return nil
}

func intPtr(v int64) *int64 {
	return &v
}

func TestServer_updateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name string
		body []byte
		want want
	}{
		{
			name: "counter test",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{"type":"counter","id":"test","delta":1}`),
		},
		{
			name: "gauge test",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{
				"type":  "gauge",
				"id":  "test",
				"value": 2.34
			}`),
		},
		{
			name: "empty type",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{
				"type":  "",
				"id":  "test",
				"value": 2.34
			}`),
		},
		{
			name: "empty metric name",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{
				"type":  "gauge",
				"id":  "",
				"value": 1
			}`),
		},
		{
			name: "unvalid counter",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{
				"type":  "counter",
				"id":  "test",
				"delta": 3.14
			}`),
		},
		{
			name: "unvalid gauge",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
			body: []byte(`{
				"type":  "gauge",
				"name":  "test",
				"value": "fail"
			}`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewBuffer(test.body))
			srv, err := NewServer(&mockStorage{}, &Config{})
			assert.NoError(t, err)
			srv.updateHandlerJSON(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestServer_getMetricHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
		storage     *metric.Metric
	}
	tests := []struct {
		name   string
		want   want
		metric metric.Metric
	}{
		{
			name: "wrong type",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
				storage:     &metric.Metric{ID: "test", MType: metric.Counter, Delta: intPtr(42)},
			},
			metric: metric.Metric{ID: "test", MType: metric.Gauge, Delta: intPtr(42)},
		},
		{
			name: "epsent metric",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
				storage:     nil,
			},
			metric: metric.Metric{ID: "test", MType: metric.Counter, Delta: intPtr(42)},
		},
		{
			name: "return metric",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
				storage:     &metric.Metric{ID: "test3", MType: metric.Counter, Delta: intPtr(42)},
			},
			metric: metric.Metric{ID: "test3", MType: metric.Counter, Delta: intPtr(42)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			err := json.NewEncoder(&body).Encode(test.metric)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/value", &body)
			w := httptest.NewRecorder()

			storage := &mockStorage{}
			if test.want.storage != nil {
				storage.Set(*test.want.storage)
			}

			srv, err := NewServer(storage, &Config{})
			assert.NoError(t, err)
			srv.getMetricHandlerJSON(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func TestServer_listMetricHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "list metrics",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/html; charset=utf-8",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			srv, err := NewServer(&mockStorage{}, &Config{})
			assert.NoError(t, err)

			srv.listMetricHandler(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

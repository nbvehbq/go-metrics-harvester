package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage/mocks"
	"github.com/stretchr/testify/assert"
)

func intPtr(v int64) *int64 {
	return &v
}

func TestServer_updateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
		callStorage bool
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
				callStorage: true,
			},
			body: []byte(`{"type":"counter","id":"test","delta":1}`),
		},
		{
			name: "gauge test",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
				callStorage: true,
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
				callStorage: false,
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
				callStorage: false,
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
				callStorage: false,
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
				callStorage: false,
			},
			body: []byte(`{
				"type":  "gauge",
				"name":  "test",
				"value": "fail"
			}`),
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockRepository(ctrl)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			if test.want.callStorage {
				m.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(metric.Metric{}, true)
			}

			req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewBuffer(test.body))
			srv, err := NewServer(m, &Config{})
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
		metric      metric.Metric
		res         bool
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: "wrong type",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
				metric:      metric.Metric{ID: "test", MType: metric.Gauge, Delta: intPtr(42)},
				res:         false,
			},
		},
		{
			name: "epsent metric",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
				metric:      metric.Metric{ID: "test", MType: metric.Counter, Delta: intPtr(42)},
				res:         false,
			},
		},
		{
			name: "return metric",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
				metric:      metric.Metric{ID: "test3", MType: metric.Counter, Delta: intPtr(42)},
				res:         true,
			},
		},
	}

	ctrl := gomock.NewController(t)
	m := mocks.NewMockRepository(ctrl)
	defer ctrl.Finish()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			err := json.NewEncoder(&body).Encode(test.want.metric)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/value", &body)
			w := httptest.NewRecorder()

			m.EXPECT().Get(gomock.Any(), test.want.metric.ID).Return(test.want.metric, test.want.res)

			srv, err := NewServer(m, &Config{})
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockRepository(ctrl)
	m.EXPECT().List(gomock.Any()).Return([]metric.Metric{}, nil)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			srv, err := NewServer(m, &Config{})
			assert.NoError(t, err)

			srv.listMetricHandler(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

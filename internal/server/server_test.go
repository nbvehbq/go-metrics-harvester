package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/stretchr/testify/assert"
)

type mockStorage struct {
	st []metric.Metric
}

func (m *mockStorage) Set(value metric.Metric) metric.Metric {
	m.st = append(m.st, value)
	return value
}

func (m *mockStorage) Get(key string) (metric.Metric, bool) {
	for _, v := range m.st {
		if v.Name == key {
			return v, true
		}
	}

	return metric.Metric{}, false
}

func (m *mockStorage) List() []metric.Metric {
	return m.st
}

func addParams(r *http.Request, params map[string]string) *http.Request {
	ctx := chi.NewRouteContext()
	for k, v := range params {
		ctx.URLParams.Add(k, v)
	}

	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
}

func TestServer_updateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name string
		want want
		req  *http.Request
	}{
		{
			name: "counter test",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
			req: addParams(httptest.NewRequest(http.MethodPost, "/update/counter/test/1", nil), map[string]string{
				"type":  "counter",
				"name":  "test",
				"value": "1",
			}),
		},
		{
			name: "gauge test",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
			req: addParams(httptest.NewRequest(http.MethodPost, "/update/{type}/{name}/{value}", nil), map[string]string{
				"type":  "gauge",
				"name":  "test",
				"value": "2.34",
			}),
		},
		{
			name: "empty type",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			req: addParams(httptest.NewRequest(http.MethodPost, "/update/{type}/{name}/{value}", nil), map[string]string{
				"type":  "",
				"name":  "test",
				"value": "2.34",
			}),
		},
		{
			name: "empty metric name",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			req: addParams(httptest.NewRequest(http.MethodPost, "/update/{type}/{name}/{value}", nil), map[string]string{
				"type":  "gauge",
				"name":  "",
				"value": "1",
			}),
		},
		{
			name: "unvalid counter",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			req: addParams(httptest.NewRequest(http.MethodPost, "/update/{type}/{name}/{value}", nil), map[string]string{
				"type":  "counter",
				"name":  "test",
				"value": "3.14",
			}),
		},
		{
			name: "unvalid gauge",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			req: addParams(httptest.NewRequest(http.MethodPost, "/update/{type}/{name}/{value}", nil), map[string]string{
				"type":  "gauge",
				"name":  "test",
				"value": "fail",
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			srv := NewServer(&mockStorage{}, &Config{})
			srv.updateHandler(w, test.req)

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
	}
	tests := []struct {
		name   string
		want   want
		req    *http.Request
		metric metric.Metric
	}{
		{
			name: "wrong type",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			req: addParams(httptest.NewRequest(http.MethodGet, "/value/{type}/{name}", nil), map[string]string{
				"type": "histogram",
				"name": "test",
			}),
			metric: metric.Metric{Name: "test", Type: metric.Counter, Value: 42},
		},
		{
			name: "epsent metric",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			req: addParams(httptest.NewRequest(http.MethodGet, "/value/{type}/{name}", nil), map[string]string{
				"type": "counter",
				"name": "test2",
			}),
			metric: metric.Metric{Name: "test", Type: metric.Counter, Value: 42},
		},
		{
			name: "return metric",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			req: addParams(httptest.NewRequest(http.MethodGet, "/value/{type}/{name}", nil), map[string]string{
				"type": "counter",
				"name": "test3",
			}),
			metric: metric.Metric{Name: "test3", Type: metric.Counter, Value: 42},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			storage := &mockStorage{}
			storage.Set(test.metric)

			srv := NewServer(storage, &Config{})
			srv.getMetricHandler(w, test.req)

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

			srv := NewServer(&mockStorage{}, &Config{})
			srv.listMetricHandler(w, req)

			res := w.Result()
			res.Body.Close()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

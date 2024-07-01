package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/stretchr/testify/assert"
)

type mockStorage struct{}

func (m *mockStorage) Set(value metric.Metric) metric.Metric {
	return value
}

func (m *mockStorage) Get(key string) (metric.Metric, bool) {
	return metric.Metric{Name: key, Type: metric.Counter, Value: 1}, true
}

func TestServer_updateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name   string
		want   want
		url string
	}{
		{
			name: "counter test",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
			url: "/update/counter/test/1",
		},
		{
			name: "gauge test",
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
			url: "/update/gauge/test/2.34",
		},
		{
			name: "empty type",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			url: "/update/test/1",
		},
		{
			name: "empty metric name",
			want: want{
				code:        404,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			url: "/update/counter/1",
		},
		{
			name: "unvalid counter",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			url: "/update/counter/test/3.14",
		},
		{
			name: "unvalid gauge",
			want: want{
				code:        400,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
			url: "/update/gauge/test/0`",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.url, nil)
			w := httptest.NewRecorder()

			srv := NewServer(&mockStorage{})
			srv.updateHandler(w, request)

			res := w.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}
